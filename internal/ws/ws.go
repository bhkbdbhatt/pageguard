// Package ws implements the small subset of RFC 6455 needed to stream scan
// events: the server handshake, unmasked text/binary writes, and masked client
// frame reads (including ping/pong/close control frames). It deliberately
// avoids third-party dependencies to keep the deploy surface minimal.
package ws

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Opcodes.
const (
	OpContinuation byte = 0x0
	OpText         byte = 0x1
	OpBinary       byte = 0x2
	OpClose        byte = 0x8
	OpPing         byte = 0x9
	OpPong         byte = 0xA
)

// Upgrader performs the server handshake.
type Upgrader struct {
	// ReadBufferSize bounds the per-connection read buffer (default 4096).
	ReadBufferSize int
	// MaxMessageSize caps an inbound message (default 1 MiB).
	MaxMessageSize int64
}

// Conn is a single WebSocket connection.
type Conn struct {
	conn net.Conn
	br   *bufio.Reader
	bw   *bufio.Writer

	maxMessageSize int64

	mu     sync.Mutex
	closed bool
}

// Upgrade hijacks the HTTP connection and completes the WebSocket handshake.
func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if !headerContainsToken(r.Header, "Connection", "upgrade") ||
		!strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, errors.New("ws: not a websocket upgrade request")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, errors.New("ws: missing Sec-WebSocket-Key")
	}
	version := r.Header.Get("Sec-WebSocket-Version")
	if version != "" && version != "13" {
		return nil, fmt.Errorf("ws: unsupported version %q", version)
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("ws: response writer does not support hijacking")
	}
	netConn, brw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}

	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + computeAccept(key) + "\r\n\r\n"
	if _, err := brw.WriteString(resp); err != nil {
		_ = netConn.Close()
		return nil, err
	}
	if err := brw.Flush(); err != nil {
		_ = netConn.Close()
		return nil, err
	}

	rs := u.ReadBufferSize
	if rs <= 0 {
		rs = 4096
	}
	maxMsg := u.MaxMessageSize
	if maxMsg <= 0 {
		maxMsg = 1 << 20
	}
	return &Conn{conn: netConn, br: brw.Reader, bw: brw.Writer, maxMessageSize: maxMsg}, nil
}

func computeAccept(key string) string {
	h := sha1.Sum([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(h[:])
}

func headerContainsToken(h http.Header, name, token string) bool {
	for _, v := range h.Values(name) {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

// WriteText sends a text message.
func (c *Conn) WriteText(payload []byte) error { return c.writeFrame(OpText, payload) }

// WriteBinary sends a binary message.
func (c *Conn) WriteBinary(payload []byte) error { return c.writeFrame(OpBinary, payload) }

// WriteClose sends a close frame and closes the connection.
func (c *Conn) WriteClose() error {
	err := c.writeFrame(OpClose, nil)
	c.close()
	return err
}

// Close closes the underlying connection.
func (c *Conn) Close() error {
	c.close()
	return nil
}

func (c *Conn) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	_ = c.conn.Close()
}

func (c *Conn) writeFrame(opcode byte, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("ws: connection closed")
	}

	header := make([]byte, 0, 10)
	header = append(header, 0x80|opcode) // FIN + opcode
	n := len(payload)
	switch {
	case n < 126:
		header = append(header, byte(n))
	case n < 65536:
		header = append(header, 126, byte(n>>8), byte(n))
	default:
		header = append(header, 127)
		var ext [8]byte
		binary.BigEndian.PutUint64(ext[:], uint64(n))
		header = append(header, ext[:]...)
	}
	if _, err := c.bw.Write(header); err != nil {
		return err
	}
	if n > 0 {
		if _, err := c.bw.Write(payload); err != nil {
			return err
		}
	}
	return c.bw.Flush()
}

// ReadMessage reads the next data message (text or binary). Control frames are
// handled transparently: ping is answered with pong, close returns io.EOF.
func (c *Conn) ReadMessage() (opcode byte, payload []byte, err error) {
	for {
		op, data, err := c.readFrame()
		if err != nil {
			return 0, nil, err
		}
		switch op {
		case OpText, OpBinary:
			return op, data, nil
		case OpPing:
			_ = c.writeFrame(OpPong, data)
		case OpPong:
			// ignore
		case OpClose:
			_ = c.writeFrame(OpClose, nil)
			return 0, nil, io.EOF
		default:
			// Continuation frames are not expected for control/event use.
		}
	}
}

func (c *Conn) readFrame() (byte, []byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(c.br, hdr[:]); err != nil {
		return 0, nil, err
	}
	fin := hdr[0]&0x80 != 0
	opcode := hdr[0] & 0x0F
	masked := hdr[1]&0x80 != 0
	length := uint64(hdr[1] & 0x7F)

	switch length {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(c.br, ext[:]); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(c.br, ext[:]); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(ext[:])
	}
	if int64(length) > c.maxMessageSize {
		return 0, nil, fmt.Errorf("ws: message too large (%d bytes)", length)
	}

	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(c.br, maskKey[:]); err != nil {
			return 0, nil, err
		}
	}
	if !fin && opcode != OpText && opcode != OpBinary {
		return 0, nil, errors.New("ws: fragmented control frame")
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(c.br, payload); err != nil {
		return 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}
	return opcode, payload, nil
}
