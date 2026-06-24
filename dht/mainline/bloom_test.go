package mainline

import (
	"encoding/hex"
	"math"
	"net"
	"testing"
)

func TestBloomFilter_InsertIP(t *testing.T) {
	bf := NewBloomFilter()

	// 192.0.2.0 - 192.0.2.255
	for i := 0; i <= 255; i++ {
		ip := net.IPv4(192, 0, 2, byte(i))
		bf.InsertIP(ip)
	}

	// 2001:DB8:: - 2001:DB8::3E7
	// Note: 3E7 in hex is 999 in decimal.
	for i := 0; i <= 999; i++ {
		ipBytes := make([]byte, 16)
		ipBytes[0], ipBytes[1] = 0x20, 0x01
		ipBytes[2], ipBytes[3] = 0x0d, 0xb8
		ipBytes[14] = byte(i >> 8)
		ipBytes[15] = byte(i & 0xff)
		ip := net.IP(ipBytes)
		bf.InsertIP(ip)
	}

	expectedHex := "f6c3f5eaa07ffd91bde89f777f26fb2bff37bdb8fb2bbaa2fd3ddde7bacfff75ee7ccbae" +
		"fe5eedb1fbfaff67f6abff5e43ddbca3fd9b9ffdf4ffd3e9dff12d1bdf59db53dbe9fa5b" +
		"7ff3b8fdfcde1afb8bedd7be2f3ee71ebbbfe93bcdeefe148246c2bc5dbff7e7efdcf24f" +
		"d8dc7adffd8fffdfddfff7a4bbeedf5cb95ce81fc7fcff1ff4ffffdfe5f7fdcbb7fd79b3" +
		"fa1fc77bfe07fff905b7b7ffc7fefeffe0b8370bb0cd3f5b7f2bd93feb4386cfdd6f7fd5" +
		"bfaf2e9ebffffeecd67adbf7c67f17efd5d75eba6ffeba7fff47a91eb1bfbb53e8abfb57" +
		"62abe8ff237279bfefbfeef5ffc5febfdfe5adffadfee1fb737ffffbfd9f6aeffeee76b6" +
		"fd8f72ef"

	actualHex := hex.EncodeToString(bf.RawBytes())
	if actualHex != expectedHex {
		t.Errorf("BloomFilter test vector mismatch.\nExpected:\n%s\nGot:\n%s", expectedHex, actualHex)
	}

	// For the 1256 inserted values the calculated estimate should be 1224.9308
	expectedEstimate := 1224.9308
	actualEstimate := bf.Estimate()

	// Compare with small tolerance
	if math.Abs(actualEstimate-expectedEstimate) > 0.0001 {
		t.Errorf("Estimate mismatch. Expected %f, got %f", expectedEstimate, actualEstimate)
	}
}
