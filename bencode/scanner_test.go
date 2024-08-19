package bencode

import (
	"bytes"
	"io"
	"testing"
)

type mockReader struct {
	data []byte
}

func (m *mockReader) Read(b []byte) (int, error) {
	if len(m.data) == 0 {
		return 0, io.EOF
	}
	n := copy(b, m.data)
	m.data = m.data[n:]
	return n, nil
}

func TestScanner_Read(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		reader   io.Reader
		expected []byte
		err      error
	}{
		{
			name:     "Empty reader",
			reader:   &mockReader{},
			expected: []byte{},
			err:      io.EOF,
		},
		{
			name:     "Read single byte",
			reader:   &mockReader{data: []byte{0x41}},
			expected: []byte{0x41},
			err:      nil,
		},
		{
			name:     "Read multiple bytes",
			reader:   &mockReader{data: []byte{0x41, 0x42, 0x43}},
			expected: []byte{0x41, 0x42, 0x43},
			err:      nil,
		},
		{
			name:     "Read partial bytes",
			reader:   &mockReader{data: []byte{0x41, 0x42, 0x43}},
			expected: []byte{0x41},
			err:      nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := &scanner{r: test.reader}
			buf := make([]byte, len(test.expected))
			n, err := s.Read(buf)

			if !bytes.Equal(buf, test.expected) {
				t.Errorf("Expected %v, but got %v", test.expected, buf)
			}

			if err != test.err {
				t.Errorf("Expected error %v, but got %v", test.err, err)
			}

			if n != len(test.expected) {
				t.Errorf("Expected %d bytes to be read, but got %d", len(test.expected), n)
			}
		})
	}
}

func TestScanner_UnreadByte(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		reader io.Reader
		err    error
	}{
		{
			name:   "Unread byte",
			reader: &mockReader{data: []byte{0x41}},
			err:    nil,
		},
		{
			name:   "Unread multiple bytes",
			reader: &mockReader{data: []byte{0x41, 0x42, 0x43}},
			err:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := &scanner{r: test.reader}
			err := s.UnreadByte()

			if err != test.err {
				t.Errorf("Expected error %v, but got %v", test.err, err)
			}
		})
	}
}
