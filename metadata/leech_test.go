package metadata

import (
	"bytes"
	"testing"

	"github.com/anacrolix/torrent/bencode"
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
