package metadata

import (
	"bytes"
	"io"
	"net"
	"reflect"
	"sync"
	"testing"

	"tgragnato.it/magnetico/bencode"
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
