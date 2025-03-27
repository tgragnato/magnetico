package metadata

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"time"

	"tgragnato.it/magnetico/v2/bencode"
	"tgragnato.it/magnetico/v2/metadata/btconn"
	"tgragnato.it/magnetico/v2/stats"
)

const MAX_METADATA_SIZE = 10 * 1024 * 1024

type rootDict struct {
	M            mDict `bencode:"m"`
	MetadataSize int   `bencode:"metadata_size"`
}

type mDict struct {
	UTMetadata int `bencode:"ut_metadata"`
}

type extDict struct {
	MsgType int `bencode:"msg_type"`
	Piece   int `bencode:"piece"`
}

type Leech struct {
	infoHash [20]byte
	peerAddr *net.TCPAddr
	ev       LeechEventHandlers

	conn     net.Conn
	clientID [20]byte

	ut_metadata                    uint8
	metadataReceived, metadataSize uint
	metadata                       []byte

	connClosed bool
}

type LeechEventHandlers struct {
	OnSuccess func(Metadata)        // must be supplied. args: metadata
	OnError   func([20]byte, error) // must be supplied. args: infohash, error
}

func NewLeech(infoHash [20]byte, peerAddr *net.TCPAddr, clientID []byte, ev LeechEventHandlers) *Leech {
	l := new(Leech)
	l.infoHash = infoHash
	l.peerAddr = peerAddr
	copy(l.clientID[:], clientID)
	l.ev = ev

	return l
}

func (l *Leech) writeAll(b []byte) error {
	for len(b) != 0 {
		n, err := l.conn.Write(b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

func (l *Leech) readExactly(n uint) ([]byte, error) {
	b := make([]byte, n)
	_, err := io.ReadFull(l.conn, b)
	return b, err
}

func (l *Leech) closeConn() {
	if l.connClosed {
		return
	}

	if err := l.conn.Close(); err != nil {
		panic("couldn't close leech connection! " + err.Error())
	}

	l.connClosed = true
}

func (l *Leech) OnError(err error) {
	l.ev.OnError(l.infoHash, err)
}

func (l *Leech) doExHandshake() error {
	err := l.writeAll([]byte("\x00\x00\x00\x1a\x14\x00d1:md11:ut_metadatai1eee"))
	if err != nil {
		return errors.New("writeAll lHandshake " + err.Error())
	}

	rExMessage, err := l.readExMessage()
	if err != nil {
		return errors.New("readExMessage " + err.Error())
	}

	// Extension Handshake has the Extension Message ID = 0x00
	if rExMessage[1] != 0 {
		return errors.New("first extension message is not an extension handshake")
	}

	rRootDict := new(rootDict)
	err = bencode.Unmarshal(rExMessage[2:], rRootDict)
	if err != nil {
		return errors.New("unmarshal rExMessage " + err.Error())
	}

	if rRootDict.MetadataSize <= 0 || rRootDict.MetadataSize >= MAX_METADATA_SIZE {
		return fmt.Errorf("metadata too big or its size is less than or equal zero")
	}

	if rRootDict.M.UTMetadata <= 0 || rRootDict.M.UTMetadata >= 255 {
		return fmt.Errorf("ut_metadata is not an uint8")
	}

	l.ut_metadata = uint8(rRootDict.M.UTMetadata) // Save the ut_metadata code the remote peer uses
	l.metadataSize = uint(rRootDict.MetadataSize)
	l.metadata = make([]byte, l.metadataSize)

	return nil
}

func (l *Leech) requestAllPieces() error {
	// Request all the pieces of metadata
	nPieces := int(math.Ceil(float64(l.metadataSize) / math.Pow(2, 14)))
	if nPieces == 0 {
		return errors.New("metadataSize is zero")
	}

	for piece := 0; piece < nPieces; piece++ {
		// __request_metadata_piece(piece)
		// ...............................
		extDictDump, err := bencode.Marshal(extDict{
			MsgType: 0,
			Piece:   piece,
		})
		if err != nil { // ASSERT
			return errors.New("marshal extDict " + err.Error())
		}

		err = l.writeAll([]byte(fmt.Sprintf(
			"%s\x14%s%s",
			toBigEndian(uint(2+len(extDictDump)), 4),
			toBigEndian(uint(l.ut_metadata), 1),
			extDictDump,
		)))
		if err != nil {
			return errors.New("writeAll piece request " + err.Error())
		}
	}

	return nil
}

// readMessage returns a BitTorrent message, sans the first 4 bytes indicating its length.
func (l *Leech) readMessage() ([]byte, error) {
	rLengthB, err := l.readExactly(4)
	if err != nil {
		return nil, errors.New("readExactly rLengthB " + err.Error())
	}

	rLength := uint(binary.BigEndian.Uint32(rLengthB))

	// Some malicious/faulty peers say that they are sending a very long
	// message, and hence causing us to run out of memory.
	// This is a crude check that does not let it happen (i.e. boundary can probably be
	// tightened a lot more.)
	if rLength > MAX_METADATA_SIZE {
		return nil, errors.New("message is longer than max allowed metadata size")
	}

	rMessage, err := l.readExactly(rLength)
	if err != nil {
		return nil, errors.New("readExactly rMessage " + err.Error())
	}

	return rMessage, nil
}

// readExMessage returns an *extension* message, sans the first 4 bytes indicating its length.
//
// It will IGNORE all non-extension messages!
func (l *Leech) readExMessage() ([]byte, error) {
	for {
		rMessage, err := l.readMessage()
		if err != nil {
			return nil, errors.New("readMessage " + err.Error())
		}

		// Every extension message has at least 2 bytes.
		if len(rMessage) < 2 {
			continue
		}

		// We are interested only in extension messages, whose first byte is always 20
		if rMessage[0] == 20 {
			return rMessage, nil
		}
	}
}

// readUmMessage returns an ut_metadata extension message, sans the first 4 bytes indicating its
// length.
//
// It will IGNORE all non-"ut_metadata extension" messages!
func (l *Leech) readUmMessage() ([]byte, error) {
	for {
		rExMessage, err := l.readExMessage()
		if err != nil {
			return nil, errors.New("readExMessage " + err.Error())
		}

		if rExMessage[1] == 0x01 {
			return rExMessage, nil
		}
	}
}

func (l *Leech) Do(deadline time.Time) {
	conn, _, peerExtensions, _, err := btconn.Dial(
		l.peerAddr,
		deadline,
		[8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x01},
		l.infoHash,
		l.clientID,
	)
	if err != nil {
		l.OnError(errors.New("btconn.Dial " + err.Error()))
		return
	}

	l.conn = conn
	defer l.closeConn()
	go stats.GetInstance().IncLeech(peerExtensions)

	err = l.doExHandshake()
	if err != nil {
		l.OnError(errors.New("doExHandshake " + err.Error()))
		return
	}

	err = l.requestAllPieces()
	if err != nil {
		l.OnError(errors.New("requestAllPieces " + err.Error()))
		return
	}

	for l.metadataReceived < l.metadataSize {
		rUmMessage, err := l.readUmMessage()
		if err != nil {
			l.OnError(errors.New("readUmMessage " + err.Error()))
			return
		}

		// Run TestDecoder() function in leech_test.go in case you have any doubts.
		rMessageBuf := bytes.NewBuffer(rUmMessage[2:])
		rExtDict := new(extDict)
		err = bencode.NewDecoder(rMessageBuf).Decode(rExtDict)
		if err != nil {
			l.OnError(errors.New("could not decode ext msg in the loop " + err.Error()))
			return
		}

		if rExtDict.MsgType == 2 { // reject
			l.OnError(fmt.Errorf("remote peer rejected sending metadata"))
			return
		}

		if rExtDict.MsgType == 1 { // data
			// Get the unread bytes!
			metadataPiece := rMessageBuf.Bytes()

			// BEP 9 explicitly states:
			//   > If the piece is the last piece of the metadata, it may be less than 16kiB. If
			//   > it is not the last piece of the metadata, it MUST be 16kiB.
			//
			// Hence...
			//   ... if the length of @metadataPiece is more than 16kiB, we err.
			if len(metadataPiece) > 16*1024 {
				l.OnError(fmt.Errorf("metadataPiece > 16kiB"))
				return
			}

			piece := rExtDict.Piece
			// metadata[piece * 2**14: piece * 2**14 + len(metadataPiece)] = metadataPiece is how it'd be done in Python
			copy(l.metadata[piece*int(math.Pow(2, 14)):piece*int(math.Pow(2, 14))+len(metadataPiece)], metadataPiece)
			l.metadataReceived += uint(len(metadataPiece))

			// ... if the length of @metadataPiece is less than 16kiB AND metadata is NOT
			// complete then we err.
			if len(metadataPiece) < 16*1024 && l.metadataReceived != l.metadataSize {
				l.OnError(fmt.Errorf("metadataPiece < 16 kiB but incomplete"))
				return
			}

			if l.metadataReceived > l.metadataSize {
				l.OnError(fmt.Errorf("metadataReceived > metadataSize"))
				return
			}
		}
	}

	// We are done with the transfer, close socket as soon as possible (i.e. NOW)
	// Avoid hitting "too many open files" error
	l.closeConn()

	extracted, err := extractMetadata(l.metadata, l.infoHash, time.Now())
	if err != nil {
		l.OnError(err)
		return
	}

	l.ev.OnSuccess(*extracted)
}
