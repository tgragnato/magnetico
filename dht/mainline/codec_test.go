package mainline

import (
	"bytes"
	"net"
	"reflect"
	"testing"

	"tgragnato.it/magnetico/v2/bencode"
)

var codecTest_validInstances = []struct {
	data []byte
	msg  Message
}{
	// ping Query:
	{
		data: []byte("d1:ad2:id20:abcdefghij0123456789e1:q4:ping1:t2:aa1:y1:qe"),
		msg: Message{
			T: []byte("aa"),
			Y: "q",
			Q: "ping",
			A: QueryArguments{
				ID: []byte("abcdefghij0123456789"),
			},
		},
	},
	// ping or announce_peer Response:
	// Also, includes NUL and EOT characters as transaction ID (`t`).
	{
		data: []byte("d1:rd2:id20:mnopqrstuvwxyz123456e1:t2:\x00\x041:y1:re"),
		msg: Message{
			T: []byte("\x00\x04"),
			Y: "r",
			R: ResponseValues{
				ID: []byte("mnopqrstuvwxyz123456"),
			},
		},
	},
	// find_node Query:
	{
		data: []byte("d1:ad2:id20:abcdefghij01234567896:target20:mnopqrstuvwxyz123456e1:q9:find_node1:t2:\x09\x0a1:y1:qe"),
		msg: Message{
			T: []byte("\x09\x0a"),
			Y: "q",
			Q: "find_node",
			A: QueryArguments{
				ID:     []byte("abcdefghij0123456789"),
				Target: []byte("mnopqrstuvwxyz123456"),
			},
		},
	},
	// find_node Response with no nodes (`nodes` key still exists):
	{
		data: []byte("d1:rd2:id20:0123456789abcdefghij5:nodes0:e1:t2:aa1:y1:re"),
		msg: Message{
			T: []byte("aa"),
			Y: "r",
			R: ResponseValues{
				ID:    []byte("0123456789abcdefghij"),
				Nodes: []CompactNodeInfo{},
			},
		},
	},
	// find_node Response with a single node:
	{
		data: []byte("d1:rd2:id20:0123456789abcdefghij5:nodes26:abcdefghijklmnopqrst\x8b\x82\x8e\xf5\x0cae1:t2:aa1:y1:re"),
		msg: Message{
			T: []byte("aa"),
			Y: "r",
			R: ResponseValues{
				ID: []byte("0123456789abcdefghij"),
				Nodes: []CompactNodeInfo{
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
				},
			},
		},
	},
	// find_node Response with 8 nodes (all the same except the very last one):
	{
		data: []byte("d1:rd2:id20:0123456789abcdefghij5:nodes208:abcdefghijklmnopqrst\x8b\x82\x8e\xf5\x0caabcdefghijklmnopqrst\x8b\x82\x8e\xf5\x0caabcdefghijklmnopqrst\x8b\x82\x8e\xf5\x0caabcdefghijklmnopqrst\x8b\x82\x8e\xf5\x0caabcdefghijklmnopqrst\x8b\x82\x8e\xf5\x0caabcdefghijklmnopqrst\x8b\x82\x8e\xf5\x0caabcdefghijklmnopqrst\x8b\x82\x8e\xf5\x0cazyxwvutsrqponmlkjihg\xf5\x8e\x82\x8b\x1b\x13e1:t2:aa1:y1:re"),
		msg: Message{
			T: []byte("aa"),
			Y: "r",
			R: ResponseValues{
				ID: []byte("0123456789abcdefghij"),
				Nodes: []CompactNodeInfo{
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("zyxwvutsrqponmlkjihg"),
						Addr: net.UDPAddr{IP: []byte("\xf5\x8e\x82\x8b"), Port: 6931, Zone: ""},
					},
				},
			},
		},
	},
	// get_peers Query:
	{
		data: []byte("d1:ad2:id20:abcdefghij01234567899:info_hash20:mnopqrstuvwxyz123456e1:q9:get_peers1:t2:aa1:y1:qe"),
		msg: Message{
			T: []byte("aa"),
			Y: "q",
			Q: "get_peers",
			A: QueryArguments{
				ID:       []byte("abcdefghij0123456789"),
				InfoHash: []byte("mnopqrstuvwxyz123456"),
			},
		},
	},
	// get_peers Response with 2 peers (`values`):
	{
		data: []byte("d1:rd2:id20:abcdefghij01234567895:token8:aoeusnth6:valuesl6:axje.u6:idhtnmee1:t2:aa1:y1:re"),
		msg: Message{
			T: []byte("aa"),
			Y: "r",
			R: ResponseValues{
				ID:    []byte("abcdefghij0123456789"),
				Token: []byte("aoeusnth"),
				Values: []CompactPeer{
					{IP: []byte("axje"), Port: 11893},
					{IP: []byte("idht"), Port: 28269},
				},
			},
		},
	},
	// get_peers Response with 2 closest nodes (`nodes`):
	{
		data: []byte("d1:rd2:id20:abcdefghij01234567895:nodes52:abcdefghijklmnopqrst\x8b\x82\x8e\xf5\x0cazyxwvutsrqponmlkjihg\xf5\x8e\x82\x8b\x1b\x135:token8:aoeusnthe1:t2:aa1:y1:re"),
		msg: Message{
			T: []byte("aa"),
			Y: "r",
			R: ResponseValues{
				ID:    []byte("abcdefghij0123456789"),
				Token: []byte("aoeusnth"),
				Nodes: []CompactNodeInfo{
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("zyxwvutsrqponmlkjihg"),
						Addr: net.UDPAddr{IP: []byte("\xf5\x8e\x82\x8b"), Port: 6931, Zone: ""},
					},
				},
			},
		},
	},
	// announce_peer Query without optional `implied_port` argument:
	{
		data: []byte("d1:ad2:id20:abcdefghij01234567899:info_hash20:mnopqrstuvwxyz1234564:porti6881e5:token8:aoeusnthe1:q13:announce_peer1:t2:aa1:y1:qe"),
		msg: Message{
			T: []byte("aa"),
			Y: "q",
			Q: "announce_peer",
			A: QueryArguments{
				ID:       []byte("abcdefghij0123456789"),
				InfoHash: []byte("mnopqrstuvwxyz123456"),
				Port:     6881,
				Token:    []byte("aoeusnth"),
			},
		},
	},
	{
		data: []byte("d1:eli201e23:A Generic Error Ocurrede1:t2:aa1:y1:ee"),
		msg: Message{
			T: []byte("aa"),
			Y: "e",
			E: Error{Code: 201, Message: []byte("A Generic Error Ocurred")},
		},
	},
	// TODO: Test Error where E.Message is an empty string, and E.Message contains invalid Unicode characters.
	// TODO: Add announce_peer Query with optional `implied_port` argument.
}

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	for i, instance := range codecTest_validInstances {
		msg := Message{}
		err := bencode.Unmarshal(instance.data, &msg)
		if err != nil {
			t.Errorf("Error while unmarshalling valid data #%d: %v", i+1, err)
			continue
		}
		if reflect.DeepEqual(msg, instance.msg) != true {
			t.Errorf("Valid data #%d unmarshalled wrong!\n\tGot     : %+v\n\tExpected: %+v",
				i+1, msg, instance.msg)
		}
	}
}

func TestMarshal(t *testing.T) {
	t.Parallel()

	for i, instance := range codecTest_validInstances {
		data, err := bencode.Marshal(instance.msg)
		if err != nil {
			t.Errorf("Error while marshalling valid msg #%d: %v", i+1, err)
		}
		if bytes.Equal(data, instance.data) != true {
			t.Errorf("Valid msg #%d marshalled wrong!\n\tGot     : %q\n\tExpected: %q",
				i+1, data, instance.data)
		}
	}
}

func TestUnmarshalCompactPeers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		binaryCompactPeers   []byte
		expectedCompactPeers CompactPeers
	}{
		{
			binaryCompactPeers: []byte{
				127, 0, 0, 1, 1, 187,
			},
			expectedCompactPeers: CompactPeers{
				{
					IP:   net.IP{127, 0, 0, 1},
					Port: 443,
				},
			},
		},
		{
			binaryCompactPeers: []byte{
				192, 168, 1, 1, 0, 80,
			},
			expectedCompactPeers: CompactPeers{
				{
					IP:   net.IP{192, 168, 1, 1},
					Port: 80,
				},
			},
		},
		{
			binaryCompactPeers: []byte{
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 123,
			},
			expectedCompactPeers: CompactPeers{
				{
					IP:   net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
					Port: 123,
				},
			},
		},
		{
			binaryCompactPeers: []byte{
				127, 0, 0, 1, 1, 187,
				192, 168, 1, 1, 0, 80,
			},
			expectedCompactPeers: CompactPeers{
				{
					IP:   net.IP{127, 0, 0, 1},
					Port: 443,
				},
				{
					IP:   net.IP{192, 168, 1, 1},
					Port: 80,
				},
			},
		},
		{
			binaryCompactPeers: []byte{
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 123,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 80,
			},
			expectedCompactPeers: CompactPeers{
				{
					IP:   net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
					Port: 123,
				},
				{
					IP:   net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
					Port: 80,
				},
			},
		},
	}
	for _, tt := range tests {
		compactPeers, err := UnmarshalCompactPeers(tt.binaryCompactPeers)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(compactPeers) != len(tt.expectedCompactPeers) {
			t.Error("Expected length of compactPeers and expectedCompactPeers to be the same")
		}
		if !reflect.DeepEqual(compactPeers[0], tt.expectedCompactPeers[0]) {
			t.Errorf("Expected %v, got %v", tt.expectedCompactPeers[0], compactPeers[0])
		}
	}
}

func TestUnmarshalBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		bytes   []byte
		wantErr bool
	}{
		{
			name:    "Empty IP:Port",
			bytes:   []byte{},
			wantErr: true,
		},
		{
			name:    "Valid IPv4:Port",
			bytes:   []byte{127, 0, 0, 1, 1, 187},
			wantErr: false,
		},
		{
			name:    "Valid IPv6:Port",
			bytes:   []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 123},
			wantErr: false,
		},
		{
			name:    "Invalid IP:Port",
			bytes:   []byte{0, 0, 0, 0, 0, 0, 0, 0},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := &CompactPeer{}
			if err := cp.UnmarshalBinary(tt.bytes); (err != nil) != tt.wantErr {
				t.Errorf("CompactPeer.UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMarshalBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		compactPeers CompactPeers
		wantBytes    []byte
		wantErr      bool
	}{
		{
			name:         "Empty CompactPeers",
			compactPeers: CompactPeers{},
			wantBytes:    []byte{},
			wantErr:      false,
		},
		{
			name: "Single CompactPeer with IPv4",
			compactPeers: CompactPeers{
				{
					IP:   net.IPv4(127, 0, 0, 1),
					Port: 443,
				},
			},
			wantBytes: []byte{127, 0, 0, 1, 1, 187},
			wantErr:   false,
		},
		{
			name: "Single CompactPeer with IPv6",
			compactPeers: CompactPeers{
				{
					IP:   net.ParseIP("::1"),
					Port: 123,
				},
			},
			wantBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 123},
			wantErr:   false,
		},
		{
			name: "Multiple CompactPeers",
			compactPeers: CompactPeers{
				{
					IP:   net.IPv4(127, 0, 0, 1),
					Port: 443,
				},
				{
					IP:   net.ParseIP("::1"),
					Port: 123,
				},
			},
			wantBytes: []byte{127, 0, 0, 1, 1, 187, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 123},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBytes, err := tt.compactPeers.MarshalBinary()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotBytes, tt.wantBytes) {
				t.Errorf("MarshalBinary() = %v, want %v", gotBytes, tt.wantBytes)
			}
		})
	}
}

func TestCompactNodeInfo_MarshalBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ID   []byte
		IP   net.IP
		Port int
		want []byte
	}{
		{
			name: "IPv4:Port",
			ID:   []byte("abcdefghijklmnopqrst"),
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 443,
			want: []byte{97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 127, 0, 0, 1, 1, 187},
		},
		{
			name: "IPv6:Port",
			ID:   []byte("abcdefghijklmnopqrst"),
			IP:   net.ParseIP("::1"),
			Port: 443,
			want: []byte{97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 187},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cni := CompactNodeInfo{
				ID:   tt.ID,
				Addr: net.UDPAddr{IP: tt.IP, Port: tt.Port},
			}
			if got := cni.MarshalBinary(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CompactNodeInfo.MarshalBinary() = %v, want %v", got, tt.want)
			}
		})
	}
}
