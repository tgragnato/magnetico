package btconn

import (
	"bytes"
	"errors"
	"io"
	"net"
	"time"
)

// The MIT License (MIT)
// Copyright (c) 2013 Cenk Alti

// Accept BitTorrent handshake from the connection. Handles encryption.
// Returns a new connection that is ready for sending/receiving BitTorrent protocol messages.
func Accept(
	conn net.Conn,
	handshakeTimeout time.Duration,
	getSKey func(sKeyHash [20]byte) (sKey []byte),
	forceEncryption bool,
	hasInfoHash func([20]byte) bool,
	ourExtensions [8]byte, ourID [20]byte) (
	encConn net.Conn, cipher CryptoMethod, peerExtensions [8]byte, peerID [20]byte, infoHash [20]byte, err error) {

	if forceEncryption && getSKey == nil {
		panic("forceEncryption && getSKey == nil")
	}

	if err = conn.SetDeadline(time.Now().Add(handshakeTimeout)); err != nil {
		return
	}

	isEncrypted := false

	// Try to do unencrypted handshake first.
	// If protocol string is not valid, try to do encrypted handshake.
	// rwConn returns the read bytes again that is read by handshake.Read1.
	var (
		buf    bytes.Buffer
		reader = io.TeeReader(conn, &buf)
	)

	peerExtensions, infoHash, err = readHandshake1(reader)
	if err != nil && getSKey != nil {
		conn = &rwConn{readWriter{io.MultiReader(&buf, conn), conn}, conn}
		mseConn := WrapConn(conn)
		err = mseConn.HandshakeIncoming(
			getSKey,
			func(provided CryptoMethod) (selected CryptoMethod) {
				if provided&RC4 != 0 {
					selected = RC4
					isEncrypted = true
				} else if (provided&PlainText != 0) && !forceEncryption {
					selected = PlainText
				}
				cipher = selected
				return
			})
		if err != nil {
			return
		}
		conn = mseConn
		peerExtensions, infoHash, err = readHandshake1(conn)
	}
	if err != nil {
		return
	}

	if forceEncryption && !isEncrypted {
		err = errors.New("encryption required but not used")
		return
	}

	if !hasInfoHash(infoHash) {
		err = errors.New("info hash mismatch")
		return
	}
	err = writeHandshake(conn, infoHash, ourID, ourExtensions)
	if err != nil {
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
	encConn = conn
	return
}
