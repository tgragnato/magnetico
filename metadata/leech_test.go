package metadata

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"reflect"
	"sync"
	"testing"

	"tgragnato.it/magnetico/v2/bencode"
)

func TestDecoder(t *testing.T) {
	t.Parallel()

	var operationInstances = []struct {
		dump    []byte
		surplus []byte
	}{
		// No Surplus
		{
			dump:    []byte("d1:md11:ut_metadatai1ee13:metadata_sizei22528ee"),
			surplus: []byte(""),
		},
		// Surplus is an ASCII string
		{
			dump:    []byte("d1:md11:ut_metadatai1ee13:metadata_sizei22528eeDENEME"),
			surplus: []byte("DENEME"),
		},
		// Surplus is a bencoded dictionary
		{
			dump:    []byte("d1:md11:ut_metadatai1ee13:metadata_sizei22528eed3:inti1337ee"),
			surplus: []byte("d3:inti1337ee"),
		},
	}

	for i, instance := range operationInstances {
		buf := bytes.NewBuffer(instance.dump)
		err := bencode.NewDecoder(buf).Decode(&struct{}{})
		if err != nil {
			t.Errorf("Couldn't decode the dump #%d! %s", i+1, err.Error())
		}

		bufSurplus := buf.Bytes()
		if !bytes.Equal(bufSurplus, instance.surplus) {
			t.Errorf("Surplus #%d is not equal to what we expected! `%s`", i+1, bufSurplus)
		}
	}
}

func TestWriteAll(t *testing.T) {
	t.Parallel()

	peer1, peer2 := net.Pipe()
	leech := &Leech{conn: peer1}
	data := []byte("Hello, World!")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		buffer := new(bytes.Buffer)
		_, err := io.Copy(buffer, peer2)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(data, buffer.Bytes()) {
			t.Errorf("Expected to read %v, but got %v", data, buffer.Bytes())
		}
		wg.Done()
	}()

	if err := leech.writeAll(data); err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	leech.closeConn()
	leech.closeConn()
	wg.Wait()
}

func TestReadExactly(t *testing.T) {
	t.Parallel()

	peer1, peer2 := net.Pipe()
	leech := &Leech{conn: peer1}
	data := []byte("Hello, World!")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		received, err := leech.readExactly(uint(len(data)))
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(data, received) {
			t.Errorf("Expected to read %v, but got %v", data, received)
		}
		wg.Done()
	}()

	if _, err := peer2.Write(data); err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if err := peer2.Close(); err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	wg.Wait()
}

func TestRequestAllPieces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		metadataSize  uint
		expectedCalls int
		expectedError bool
	}{
		{
			name:          "Single piece",
			metadataSize:  16 * 1024,
			expectedCalls: 1,
			expectedError: false,
		},
		{
			name:          "Multiple pieces",
			metadataSize:  32 * 1024,
			expectedCalls: 2,
			expectedError: false,
		},
		{
			name:          "Exact multiple pieces",
			metadataSize:  48 * 1024,
			expectedCalls: 3,
			expectedError: false,
		},
		{
			name:          "Zero size",
			metadataSize:  0,
			expectedCalls: 0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer1, peer2 := net.Pipe()
			leech := &Leech{conn: peer1, metadataSize: tt.metadataSize}

			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				buffer := new(bytes.Buffer)
				_, err := io.Copy(buffer, peer2)
				if err != nil {
					t.Error(err)
				}

				// Check the number of requests sent
				requests := 0
				for buffer.Len() > 0 {
					length := binary.BigEndian.Uint32(buffer.Next(4))
					buffer.Next(int(length))
					requests++
				}

				if requests != tt.expectedCalls {
					t.Errorf("Expected %d requests, but got %d", tt.expectedCalls, requests)
				}
			}()

			err := leech.requestAllPieces()
			if (err != nil) != tt.expectedError {
				t.Errorf("Expected error: %v, got: %v", tt.expectedError, err)
			}

			leech.closeConn()
			wg.Wait()
		})
	}
}

func TestReadMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         []byte
		expected      []byte
		expectedError bool
	}{
		{
			name:          "Valid message",
			input:         append([]byte{0, 0, 0, 5}, []byte("hello")...),
			expected:      []byte("hello"),
			expectedError: false,
		},
		{
			name:          "Message too long",
			input:         append([]byte{0xff, 0xff, 0xff, 0xff}, []byte("hello")...),
			expected:      nil,
			expectedError: true,
		},
		{
			name:          "Incomplete message",
			input:         []byte{0, 0, 0, 5, 'h', 'e'},
			expected:      nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer1, peer2 := net.Pipe()
			leech := &Leech{conn: peer1}

			go func() {
				if _, err := peer2.Write(tt.input); err != nil {
					t.Error(err)
				}
				peer2.Close()
			}()

			result, err := leech.readMessage()
			if (err != nil) != tt.expectedError {
				t.Errorf("Expected error: %v, got: %v", tt.expectedError, err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected: %v, got: %v", tt.expected, result)
			}
		})
	}
}

func TestReadExMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         []byte
		expected      []byte
		expectedError bool
	}{
		{
			name:          "Valid extension message",
			input:         append([]byte{0, 0, 0, 7}, []byte{20, 1, 'h', 'e', 'l', 'l', 'o'}...),
			expected:      []byte{20, 1, 'h', 'e', 'l', 'l', 'o'},
			expectedError: false,
		},
		{
			name:          "Non-extension message",
			input:         append([]byte{0, 0, 0, 6}, []byte{19, 1, 'h', 'e', 'l', 'l', 'o'}...),
			expected:      nil,
			expectedError: true,
		},
		{
			name:          "Incomplete message",
			input:         []byte{0, 0, 0, 6, 20, 1, 'h'},
			expected:      nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer1, peer2 := net.Pipe()
			leech := &Leech{conn: peer1}

			go func() {
				if _, err := peer2.Write(tt.input); err != nil {
					t.Error(err)
				}
				peer2.Close()
			}()

			result, err := leech.readExMessage()
			if (err != nil) != tt.expectedError {
				t.Errorf("Expected error: %v, got: %v", tt.expectedError, err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected: %v, got: %v", tt.expected, result)
			}
		})
	}
}

func TestReadUmMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         []byte
		expected      []byte
		expectedError bool
	}{
		{
			name:          "Valid ut_metadata message",
			input:         append([]byte{0, 0, 0, 7}, []byte{20, 1, 'h', 'e', 'l', 'l', 'o'}...),
			expected:      []byte{20, 1, 'h', 'e', 'l', 'l', 'o'},
			expectedError: false,
		},
		{
			name:          "Non-ut_metadata extension message",
			input:         append([]byte{0, 0, 0, 7}, []byte{20, 2, 'h', 'e', 'l', 'l', 'o'}...),
			expected:      nil,
			expectedError: true,
		},
		{
			name:          "Incomplete ut_metadata message",
			input:         []byte{0, 0, 0, 7, 20, 1, 'h'},
			expected:      nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer1, peer2 := net.Pipe()
			leech := &Leech{conn: peer1}

			go func() {
				if _, err := peer2.Write(tt.input); err != nil {
					t.Error(err)
				}
				peer2.Close()
			}()

			result, err := leech.readUmMessage()
			if (err != nil) != tt.expectedError {
				t.Errorf("Expected error: %v, got: %v", tt.expectedError, err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected: %v, got: %v", tt.expected, result)
			}
		})
	}
}

func TestDoExHandshake(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          []byte
		expectedOutput []byte
		expectedError  bool
	}{
		{
			name:           "Valid handshake",
			input:          append([]byte{0, 0, 0, 49}, []byte{20, 0, 'd', '1', ':', 'm', 'd', '1', '1', ':', 'u', 't', '_', 'm', 'e', 't', 'a', 'd', 'a', 't', 'a', 'i', '1', 'e', 'e', '1', '3', ':', 'm', 'e', 't', 'a', 'd', 'a', 't', 'a', '_', 's', 'i', 'z', 'e', 'i', '2', '2', '5', '2', '8', 'e', 'e'}...),
			expectedOutput: []byte{0, 0, 0, 26, 20, 0, 100, 49, 58, 109, 100, 49, 49, 58, 117, 116, 95, 109, 101, 116, 97, 100, 97, 116, 97, 105, 49, 101, 101, 101},
			expectedError:  false,
		},
		{
			name:           "Invalid extension message ID",
			input:          append([]byte{0, 0, 0, 50}, []byte{20, 1, 'd', '1', ':', 'm', 'd', '1', '1', ':', 'u', 't', '_', 'm', 'e', 't', 'a', 'd', 'a', 't', 'a', 'i', '1', 'e', 'e', 'e', '1', '3', ':', 'm', 'e', 't', 'a', 'd', 'a', 't', 'a', '_', 's', 'i', 'z', 'e', 'i', '2', '2', '5', '2', '8', 'e', 'e'}...),
			expectedOutput: []byte{0, 0, 0, 26, 20, 0, 100, 49, 58, 109, 100, 49, 49, 58, 117, 116, 95, 109, 101, 116, 97, 100, 97, 116, 97, 105, 49, 101, 101, 101},
			expectedError:  true,
		},
		{
			name:           "Invalid metadata size",
			input:          append([]byte{0, 0, 0, 45}, []byte{20, 0, 'd', '1', ':', 'm', 'd', '1', '1', ':', 'u', 't', '_', 'm', 'e', 't', 'a', 'd', 'a', 't', 'a', 'i', '1', 'e', 'e', '1', '3', ':', 'm', 'e', 't', 'a', 'd', 'a', 't', 'a', '_', 's', 'i', 'z', 'e', 'i', '0', 'e', 'e'}...),
			expectedOutput: []byte{0, 0, 0, 26, 20, 0, 100, 49, 58, 109, 100, 49, 49, 58, 117, 116, 95, 109, 101, 116, 97, 100, 97, 116, 97, 105, 49, 101, 101, 101},
			expectedError:  true,
		},
		{
			name:           "Invalid ut_metadata",
			input:          append([]byte{0, 0, 0, 50}, []byte{20, 0, 'd', '1', ':', 'm', 'd', '1', '1', ':', 'u', 't', '_', 'm', 'e', 't', 'a', 'd', 'a', 't', 'a', 'i', '0', 'e', 'e', 'e', '1', '3', ':', 'm', 'e', 't', 'a', 'd', 'a', 't', 'a', '_', 's', 'i', 'z', 'e', 'i', '2', '2', '5', '2', '8', 'e', 'e'}...),
			expectedOutput: []byte{0, 0, 0, 26, 20, 0, 100, 49, 58, 109, 100, 49, 49, 58, 117, 116, 95, 109, 101, 116, 97, 100, 97, 116, 97, 105, 49, 101, 101, 101},
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer1, peer2 := net.Pipe()
			leech := &Leech{conn: peer1}
			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				var buffer bytes.Buffer
				if _, err := io.Copy(&buffer, peer2); err != nil {
					t.Error(err)
				}
				if !reflect.DeepEqual(buffer.Bytes(), tt.expectedOutput) {
					t.Errorf("Expected output: %v, got: %v", tt.expectedOutput, buffer.Bytes())
				}
			}()

			go func() {
				defer wg.Done()
				if _, err := peer2.Write(tt.input); err != nil {
					t.Error(err)
				}
			}()

			err := leech.doExHandshake()
			if (err != nil) != tt.expectedError {
				t.Errorf("Expected error: %v, got: %v", tt.expectedError, err)
			}
			leech.closeConn()

			wg.Wait()
		})
	}
}
