package btconn

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rc4"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	mrand "math/rand/v2"
)

// The MIT License (MIT)
// Copyright (c) 2013 Cenk Alti

var (
	pBytes = []byte{255, 255, 255, 255, 255, 255, 255, 255, 201, 15, 218, 162, 33, 104, 194, 52, 196, 198, 98, 139, 128, 220, 28, 209, 41, 2, 78, 8, 138, 103, 204, 116, 2, 11, 190, 166, 59, 19, 155, 34, 81, 74, 8, 121, 142, 52, 4, 221, 239, 149, 25, 179, 205, 58, 67, 27, 48, 43, 10, 109, 242, 95, 20, 55, 79, 225, 53, 109, 109, 81, 194, 69, 228, 133, 181, 118, 98, 94, 126, 198, 244, 76, 66, 233, 166, 58, 54, 33, 0, 0, 0, 0, 0, 9, 5, 99}
	p      = new(big.Int)
	g      = big.NewInt(2)
	vc     = make([]byte, 8)
)

func init() { p.SetBytes(pBytes) }

// CryptoMethod is 32-bit bitfield each bit representing a single crypto method.
type CryptoMethod uint32

// Crypto methods
const (
	PlainText CryptoMethod = 1 << iota
	RC4
)

func (c CryptoMethod) String() string {
	switch c {
	case PlainText:
		return "PlainText"
	case RC4:
		return "RC4"
	default:
		return "unknown"
	}
}

// Stream wraps a io.ReadWriter that automatically does encrypt/decrypt on read/write.
type Stream struct {
	raw io.ReadWriter
	r   *cipher.StreamReader
	w   *cipher.StreamWriter
	r2  io.Reader
}

// NewStream returns a new Stream. You must call HandshakeIncoming or
// HandshakeOutgoing methods before using Read/Write methods.
// If any error happens during the handshake underlying io.ReadWriter will be closed if it implements io.Closer.
func NewStream(rw io.ReadWriter) *Stream { return &Stream{raw: rw} }

// Read from underlying io.ReadWriter, decrypt bytes and put into p.
func (s *Stream) Read(p []byte) (n int, err error) { return s.r2.Read(p) }

// Encrypt bytes in p and write into underlying io.ReadWriter.
func (s *Stream) Write(p []byte) (n int, err error) { return s.w.Write(p) }

// HandshakeOutgoing initiates MSE handshake for outgoing stream.
//
// sKey is stream identifier key. Same key must be used at the other side of the stream, otherwise handshake fails.
//
// cryptoProvide is a bitfield for specifying supported encryption methods.
//
// initialPayload is going to be sent along with handshake. It may be nil if you want to wait for the encryption negotiation.
func (s *Stream) HandshakeOutgoing(sKey []byte, cryptoProvide CryptoMethod, initialPayload []byte) (selected CryptoMethod, err error) {
	if cryptoProvide == 0 {
		err = errors.New("no crypto methods are provided")
		return
	}
	if len(initialPayload) > math.MaxUint16 {
		err = errors.New("initial payload is too big")
		return
	}

	writeBuf := bytes.NewBuffer(make([]byte, 0, 96+512))

	Xa, Ya := keyPair()

	// Step 1 | A->B: Diffie Hellman Ya, PadA
	writeBuf.Write(bytesWithPad(Ya))
	padA, err := padRandom()
	if err != nil {
		return
	}
	writeBuf.Write(padA)
	_, err = writeBuf.WriteTo(s.raw)
	if err != nil {
		return
	}

	// Step 2 | B->A: Diffie Hellman Yb, PadB
	b := make([]byte, 96+512)
	firstRead, err := io.ReadAtLeast(s.raw, b, 96)
	if err != nil {
		return
	}
	Yb := new(big.Int)
	Yb.SetBytes(b[:96])
	S := Yb.Exp(Yb, Xa, p)
	err = s.initRC4("keyA", "keyB", S, sKey)
	if err != nil {
		return
	}

	// Step 3 | A->B: HASH('req1', S), HASH('req2', SKEY) xor HASH('req3', S), ENCRYPT(VC, crypto_provide, len(PadC), PadC, len(IA)), ENCRYPT(IA)
	hashS, hashSKey := hashes(S, sKey)
	padC, err := padZero()
	if err != nil {
		return
	}
	writeBuf.Write(hashS)
	writeBuf.Write(hashSKey)
	writeBuf.Write(vc)
	_ = binary.Write(writeBuf, binary.BigEndian, cryptoProvide)
	if len(padC) > math.MaxUint16 {
		err = errors.New("padC is too big")
		return
	}
	_ = binary.Write(writeBuf, binary.BigEndian, uint16(len(padC)))
	writeBuf.Write(padC)
	if len(initialPayload) > math.MaxUint16 {
		err = errors.New("initial payload is too big")
		return
	}
	_ = binary.Write(writeBuf, binary.BigEndian, uint16(len(initialPayload)))
	writeBuf.Write(initialPayload)
	encBytes := writeBuf.Bytes()[40:]
	s.w.S.XORKeyStream(encBytes, encBytes) // RC4
	_, err = writeBuf.WriteTo(s.raw)
	if err != nil {
		return
	}

	// Step 4 | B->A: ENCRYPT(VC, crypto_select, len(padD), padD), ENCRYPT2(Payload Stream)
	vcEnc := make([]byte, 8)
	s.r.S.XORKeyStream(vcEnc, vc)
	err = s.readSync(vcEnc, 616-firstRead)
	if err != nil {
		return
	}
	err = binary.Read(s.r, binary.BigEndian, &selected)
	if err != nil {
		return
	}
	if selected == 0 {
		err = errors.New("none of the provided methods are accepted")
		return
	}
	if !isPowerOfTwo(uint32(selected)) {
		err = fmt.Errorf("invalid crypto selected: %d", selected)
		return
	}
	if (selected & cryptoProvide) == 0 {
		err = fmt.Errorf("selected crypto was not provided: %d", selected)
		return
	}
	var lenPadD uint16
	err = binary.Read(s.r, binary.BigEndian, &lenPadD)
	if err != nil {
		return
	}
	_, err = io.CopyN(io.Discard, s.r, int64(lenPadD))
	if err != nil {
		return
	}
	s.updateCipher(selected)
	s.r2 = s.r

	return
	// Step 5 | A->B: ENCRYPT2(Payload Stream)
}

// HandshakeIncoming initiates MSE handshake for incoming stream.
//
// getSKey must return the correct stream identifier for given sKeyHash.
// sKeyHash can be calculated with mse.HashSKey function.
// If there is no matching sKeyHash in your application, you must return nil.
//
// cryptoSelect is a function that takes provided methods as a bitfield and returns the selected crypto method.
// Function may return zero value that means none of the provided methods are selected and handshake fails.
//
// payloadIn is a buffer for writing initial payload that is coming along with the handshake from the initiator of the handshake.
// If initial payload does not fit into payloadIn, handshake returns io.ErrShortBuffer.
//
// lenPayloadIn is length of the data read into payloadIn.
//
// processPayloadIn is an optional function that processes incoming initial payload and generate outgoing initial payload.
// If this function returns an error, handshake fails.
func (s *Stream) HandshakeIncoming(
	getSKey func(sKeyHash [20]byte) (sKey []byte),
	cryptoSelect func(provided CryptoMethod) (selected CryptoMethod)) (err error) {
	writeBuf := bytes.NewBuffer(make([]byte, 0, 96+512))

	Xb, Yb := keyPair()

	// Step 1 | A->B: Diffie Hellman Ya, PadA
	b := make([]byte, 96+512)
	firstRead, err := io.ReadAtLeast(s.raw, b, 96)
	if err != nil {
		return
	}
	Ya := new(big.Int)
	Ya.SetBytes(b[:96])
	S := Ya.Exp(Ya, Xb, p)

	// Step 2 | B->A: Diffie Hellman Yb, PadB
	writeBuf.Write(bytesWithPad(Yb))
	padB, err := padRandom()
	if err != nil {
		return
	}
	writeBuf.Write(padB)
	_, err = writeBuf.WriteTo(s.raw)
	if err != nil {
		return
	}

	// Step 3 | A->B: HASH('req1', S), HASH('req2', SKEY) xor HASH('req3', S), ENCRYPT(VC, crypto_provide, len(PadC), PadC, len(IA)), ENCRYPT(IA)
	req1 := hashInt("req1", S)
	err = s.readSync(req1, 628-firstRead)
	if err != nil {
		return
	}
	var hashRead [20]byte
	_, err = io.ReadFull(s.raw, hashRead[:])
	if err != nil {
		return
	}
	req3 := hashInt("req3", S)
	for i := 0; i < sha1.Size; i++ {
		hashRead[i] ^= req3[i]
	}
	sKey := getSKey(hashRead)
	if sKey == nil {
		err = errors.New("invalid SKEY hash")
		return
	}
	err = s.initRC4("keyB", "keyA", S, sKey)
	if err != nil {
		return
	}
	vcRead := make([]byte, 8)
	_, err = io.ReadFull(s.r, vcRead)
	if err != nil {
		return
	}
	if !bytes.Equal(vcRead, vc) {
		err = fmt.Errorf("invalid VC: %s", hex.EncodeToString(vcRead))
		return
	}
	var cryptoProvide CryptoMethod
	err = binary.Read(s.r, binary.BigEndian, &cryptoProvide)
	if err != nil {
		return
	}
	if cryptoProvide == 0 {
		err = errors.New("no crypto methods are provided")
		return
	}
	selected := cryptoSelect(cryptoProvide)
	if selected == 0 {
		err = errors.New("none of the provided methods are accepted")
		return
	}
	if !isPowerOfTwo(uint32(selected)) {
		err = fmt.Errorf("invalid crypto selected: %d", selected)
		return
	}
	if (selected & cryptoProvide) == 0 {
		err = fmt.Errorf("selected crypto is not provided: %d", selected)
		return
	}
	var lenPadC uint16
	err = binary.Read(s.r, binary.BigEndian, &lenPadC)
	if err != nil {
		return
	}
	_, err = io.CopyN(io.Discard, s.r, int64(lenPadC))
	if err != nil {
		return
	}
	var lenIA uint16
	err = binary.Read(s.r, binary.BigEndian, &lenIA)
	if err != nil {
		return
	}
	IA := bytes.NewBuffer(make([]byte, 0, lenIA))
	_, err = io.CopyN(IA, s.r, int64(lenIA))
	if err != nil {
		return
	}

	// Step 4 | B->A: ENCRYPT(VC, crypto_select, len(padD), padD), ENCRYPT2(Payload Stream)
	writeBuf.Write(vc)
	_ = binary.Write(writeBuf, binary.BigEndian, selected)
	padD, err := padZero()
	if err != nil {
		return
	}
	if len(padD) > math.MaxUint16 {
		err = errors.New("padD is too big")
		return
	}
	_ = binary.Write(writeBuf, binary.BigEndian, uint16(len(padD)))
	writeBuf.Write(padD)

	_, err = writeBuf.WriteTo(s.w)
	if err != nil {
		return
	}

	s.updateCipher(selected)
	s.r2 = io.MultiReader(IA, s.r)
	return
	// Step 5 | A->B: ENCRYPT2(Payload Stream)
}

func (s *Stream) initRC4(encKey, decKey string, S *big.Int, sKey []byte) error { //nolint:gocritic
	cipherEnc, err := rc4.NewCipher(rc4Key(encKey, S, sKey))
	if err != nil {
		return err
	}
	cipherDec, err := rc4.NewCipher(rc4Key(decKey, S, sKey))
	if err != nil {
		return err
	}
	var buf [1024]byte
	discard := buf[:]
	cipherEnc.XORKeyStream(discard, discard)
	cipherDec.XORKeyStream(discard, discard)
	s.w = &cipher.StreamWriter{S: cipherEnc, W: s.raw}
	s.r = &cipher.StreamReader{S: cipherDec, R: s.raw}
	return nil
}

func (s *Stream) updateCipher(selected CryptoMethod) {
	switch selected {
	case RC4:
	case PlainText:
		s.r = &cipher.StreamReader{S: plainTextCipher{}, R: s.raw}
		s.w = &cipher.StreamWriter{S: plainTextCipher{}, W: s.raw}
	}
}

func (s *Stream) readSync(key []byte, max int) error {
	var readBuf bytes.Buffer
	if _, err := io.CopyN(&readBuf, s.raw, int64(len(key))); err != nil {
		return err
	}
	max -= len(key)
	for {
		if bytes.Equal(readBuf.Bytes(), key) {
			return nil
		}
		if max <= 0 {
			return errors.New("sync point is not found")
		}
		if _, err := io.CopyN(&readBuf, s.raw, 1); err != nil {
			return err
		}
		max--
		if _, err := io.CopyN(io.Discard, &readBuf, 1); err != nil {
			return err
		}
	}
}

func privateKey() *big.Int {
	b := make([]byte, 20)
	_, err := rand.Read(b)
	if err != nil {
		for i := 0; i < 20; i++ {
			b[i] = byte(mrand.IntN(256))
		}
	}
	var n big.Int
	return n.SetBytes(b)
}

func publicKey(private *big.Int) *big.Int {
	var n big.Int
	return n.Exp(g, private, p)
}

func keyPair() (private, public *big.Int) {
	private = privateKey()
	public = publicKey(private)
	return
}

// bytesWithPad adds padding in front of the bytes to fill 96 bytes.
func bytesWithPad(key *big.Int) []byte {
	b := key.Bytes()
	pad := 96 - len(b)
	if pad > 0 {
		b = make([]byte, 96)
		copy(b[pad:], key.Bytes())
	}
	return b
}

func isPowerOfTwo(x uint32) bool { return (x != 0) && ((x & (x - 1)) == 0) }

func hashes(S *big.Int, sKey []byte) (hashS, hashSKey []byte) { // nolint:gocritic
	req1 := hashInt("req1", S)
	req2 := HashSKey(sKey)
	req3 := hashInt("req3", S)
	for i := 0; i < sha1.Size; i++ {
		req3[i] ^= req2[i]
	}
	return req1, req3
}

func hashInt(prefix string, i *big.Int) []byte {
	h := sha1.New()
	_, _ = h.Write([]byte(prefix))
	_, _ = h.Write(bytesWithPad(i))
	return h.Sum(nil)
}

// HashSKey returns the hash of key.
func HashSKey(key []byte) [20]byte {
	var sum [20]byte
	h := sha1.New()
	_, _ = h.Write([]byte("req2"))
	_, _ = h.Write(key)
	copy(sum[:], h.Sum(nil))
	return sum
}

func rc4Key(prefix string, S *big.Int, sKey []byte) []byte { // nolint:gocritic
	h := sha1.New()
	_, _ = h.Write([]byte(prefix))
	_, _ = h.Write(bytesWithPad(S))
	_, _ = h.Write(sKey)
	return h.Sum(nil)
}

func padRandom() ([]byte, error) {
	b, err := padZero()
	if err != nil {
		return nil, err
	}
	_, err = rand.Read(b)
	if err != nil {
		for i := 0; i < len(b); i++ {
			b[i] = byte(mrand.IntN(256))
		}
	}
	return b, nil
}

func padZero() ([]byte, error) {
	padLen, err := rand.Int(rand.Reader, big.NewInt(512))
	if err != nil {
		return nil, err
	}
	return make([]byte, int(padLen.Int64())), nil
}

type plainTextCipher struct{}

func (plainTextCipher) XORKeyStream(dst, src []byte) { copy(dst, src) }
