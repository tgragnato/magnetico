package metadata

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	mrand "math/rand"

	"github.com/tgragnato/magnetico/metainfo"
	"github.com/tgragnato/magnetico/persistence"
)

func totalSize(files []persistence.File) (uint64, error) {
	var totalSize uint64
	if len(files) == 0 {
		return 0, errors.New("no files would be persisted")
	}

	for _, file := range files {
		if file.Size < 0 {
			return 0, errors.New("file size less than zero")
		}

		totalSize += uint64(file.Size)
	}
	return totalSize, nil
}

func validateInfo(info *metainfo.Info) error {
	if len(info.Pieces)%20 != 0 {
		return errors.New("pieces has invalid length")
	}
	if info.PieceLength == 0 {
		return errors.New("zero piece length")
	}
	if int((info.TotalLength()+info.PieceLength-1)/info.PieceLength) != info.NumPieces() {
		return errors.New("piece count and file lengths are at odds")
	}
	return nil
}

func randomID() []byte {
	prefix := []byte(PeerPrefix)
	var rando []byte

	peace := PeerIDLength - len(prefix)
	for i := peace; i > 0; i-- {
		rando = append(rando, randomDigit())
	}

	return append(prefix, rando...)
}

// randomDigit as byte (ASCII code range 0-9 digits)
func randomDigit() byte {
	b := make([]byte, 1)
	_, err := rand.Read(b)
	if err != nil {
		b[0] = byte(mrand.Intn(256))
	}
	return (b[0] % 10) + '0'
}

func toBigEndian(i uint, n int) []byte {
	b := make([]byte, n)
	switch n {
	case 1:
		b = []byte{byte(i)}

	case 2:
		binary.BigEndian.PutUint16(b, uint16(i))

	case 4:
		binary.BigEndian.PutUint32(b, uint32(i))

	default:
		panic("n must be 1, 2, or 4!")
	}

	if len(b) != n {
		panic(fmt.Sprintf("postcondition failed: len(b) != n in intToBigEndian (i %d, n %d, len b %d, b %s)", i, n, len(b), b))
	}

	return b
}
