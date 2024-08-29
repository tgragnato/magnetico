package btconn

import (
	"bytes"
	"context"
	"errors"
	"net"
	"time"
)

// The MIT License (MIT)
// Copyright (c) 2013 Cenk Alti

// Dial new connection to the address. Does the BitTorrent protocol handshake.
// Handles encryption. May try to connect again if encryption does not match with given setting.
// Returns a net.Conn that is ready for sending/receiving BitTorrent peer protocol messages.
func Dial(
	addr net.Addr,
	deadline time.Time,
	enableEncryption,
	forceEncryption bool,
	ourExtensions [8]byte,
	ih [20]byte,
	ourID [20]byte) (
	conn net.Conn, cipher CryptoMethod, peerExtensions [8]byte, peerID [20]byte, err error) {
	// First connection - Connecting to peer
	dialer := net.Dialer{Deadline: deadline}
	conn, err = dialer.DialContext(context.Background(), addr.Network(), addr.String())
	if err != nil {
		return
	}
	defer func(conn net.Conn) {
		if err != nil {
			conn.Close()
		}
	}(conn)

	// Write first part of BitTorrent handshake to a buffer because we will use it in both encrypted and unencrypted handshake.
	out := bytes.NewBuffer(make([]byte, 0, 68))
	err = writeHandshake(out, ih, ourID, ourExtensions)
	if err != nil {
		return
	}

	// Handshake must be completed in allowed duration.
	if err = conn.SetDeadline(deadline); err != nil {
		return
	}

	if enableEncryption {
		sKey := make([]byte, 20)
		copy(sKey, ih[:])

		provide := RC4
		if !forceEncryption {
			provide |= PlainText
		}

		// Try encryption handshake
		encConn := WrapConn(conn)
		cipher, err = encConn.HandshakeOutgoing(sKey, provide, out.Bytes())
		if err != nil {
			if forceEncryption {
				return
			}

			// Close current connection and try again without encryption
			conn.Close()
			conn, err = dialer.DialContext(context.Background(), addr.Network(), addr.String())
			if err != nil {
				return
			}
			defer func(conn net.Conn) {
				if err != nil {
					conn.Close()
				}
			}(conn)

			// Send BT handshake
			if err = conn.SetDeadline(deadline); err != nil {
				return
			}
			if _, err = conn.Write(out.Bytes()); err != nil {
				return
			}
		} else {
			conn = encConn
		}
	} else {
		// Send BT handshake
		_, err = conn.Write(out.Bytes())
		if err != nil {
			return
		}
	}

	// Read BT handshake
	var ihRead [20]byte
	peerExtensions, ihRead, err = readHandshake1(conn)
	if err != nil {
		return
	}
	if ihRead != ih {
		err = errors.New("invalid infohash")
		return
	}

	peerID, err = readHandshake2(conn)
	if err != nil {
		return
	}
	if peerID == ourID {
		err = errors.New("peerID matches ourID")
		return
	}
	return
}
