// Package mainline: Bloom filter implementation
//
// BEP-33 (DHT Scrapes) mandates a specific Bloom filter algorithm:
//  - k = 2, m = 256*8 (2048 bits)
//  - insertion: SHA-1(ip) and use the first 4 bytes to derive two indices
//  - exact 256-byte representation for network bencode fields (BFsd/BFpe)

package mainline

import (
	"crypto/sha1"
	"errors"
	"math"
	"net"
)

type BloomFilter struct {
	m     uint
	k     uint
	bytes []byte
}

func NewBloomFilter() *BloomFilter {
	return &BloomFilter{
		m:     2048,
		k:     2,
		bytes: make([]byte, 256),
	}
}

func (bf *BloomFilter) InsertIP(ip net.IP) {
	canonical := ip.To4()
	if canonical == nil {
		canonical = ip.To16()
	}
	if canonical == nil {
		return
	}

	hash := sha1.Sum(canonical)

	index1 := uint(hash[0]) | uint(hash[1])<<8
	index2 := uint(hash[2]) | uint(hash[3])<<8

	index1 %= bf.m
	index2 %= bf.m

	bf.bytes[index1/8] |= 0x01 << (index1 % 8)
	bf.bytes[index2/8] |= 0x01 << (index2 % 8)
}

func (bf *BloomFilter) Estimate() float64 {
	var zeroBits uint = 0
	for _, b := range bf.bytes {
		for i := 0; i < 8; i++ {
			if (b & (0x01 << i)) == 0 {
				zeroBits++
			}
		}
	}
	if zeroBits == 0 {
		return float64(bf.m)
	}
	c := float64(zeroBits)
	if c > float64(bf.m-1) {
		c = float64(bf.m - 1)
	}

	size := math.Log(c/float64(bf.m)) / (float64(bf.k) * math.Log(1.0-1.0/float64(bf.m)))
	return size
}

func (bf *BloomFilter) RawBytes() []byte {
	return bf.bytes
}

func (bf *BloomFilter) MarshalBencode() ([]byte, error) {
	if bf == nil || len(bf.bytes) != 256 {
		return []byte("0:"), nil
	}
	// Bencode representation of a 256-byte string is "256:" followed by the bytes
	res := make([]byte, 0, 4+256)
	res = append(res, []byte("256:")...)
	res = append(res, bf.bytes...)
	return res, nil
}

func (bf *BloomFilter) UnmarshalBencode(b []byte) error {
	if len(b) == 0 {
		return errors.New("empty bencode for bloom filter")
	}
	colonIdx := -1
	for i, c := range b {
		if c == ':' {
			colonIdx = i
			break
		}
	}
	if colonIdx == -1 {
		return errors.New("invalid bencode string (missing colon)")
	}
	data := b[colonIdx+1:]
	if len(data) != 256 {
		return errors.New("bloom filter must be 256 bytes")
	}
	bf.m = 2048
	bf.k = 2
	bf.bytes = make([]byte, 256)
	copy(bf.bytes, data)
	return nil
}
