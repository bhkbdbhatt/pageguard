package ws

import (
	"bufio"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUpgradeHandshakeAndEcho(t *testing.T) {
	up := &Upgrader{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("read: %v", err)
			return
		}
		if err := conn.WriteText(append([]byte("echo:"), msg...)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")
	netConn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer netConn.Close()

	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
	req := "GET / HTTP/1.1\r\n" +
		"Host: " + addr + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"
	if _, err := netConn.Write([]byte(req)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	br := bufio.NewReader(netConn)
	status, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if !strings.Contains(status, "101") {
		t.Fatalf("status = %q", strings.TrimSpace(status))
	}
	var accept string
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatalf("read header: %v", err)
		}
		if line == "\r\n" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "sec-websocket-accept:") {
			accept = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
	}
	if accept != computeAccept(key) {
		t.Errorf("accept = %q, want %q", accept, computeAccept(key))
	}

	// Send a masked text frame "hi".
	if err := writeMaskedText(netConn, []byte("hi")); err != nil {
		t.Fatalf("write frame: %v", err)
	}

	// Read the server's text frame.
	opcode, payload, err := readFrame(br)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if opcode != OpText {
		t.Errorf("opcode = %d", opcode)
	}
	if string(payload) != "echo:hi" {
		t.Errorf("payload = %q", payload)
	}
}

func writeMaskedText(w io.Writer, payload []byte) error {
	mask := [4]byte{0x37, 0xfa, 0x21, 0x3d}
	frame := []byte{0x80 | OpText, 0x80 | byte(len(payload)), mask[0], mask[1], mask[2], mask[3]}
	for i, b := range payload {
		frame = append(frame, b^mask[i%4])
	}
	_, err := w.Write(frame)
	return err
}

func readFrame(br *bufio.Reader) (byte, []byte, error) {
	var h [2]byte
	if _, err := io.ReadFull(br, h[:]); err != nil {
		return 0, nil, err
	}
	opcode := h[0] & 0x0f
	length := int(h[1] & 0x7f)
	if length == 126 {
		var ext [2]byte
		if _, err := io.ReadFull(br, ext[:]); err != nil {
			return 0, nil, err
		}
		length = int(ext[0])<<8 | int(ext[1])
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(br, payload); err != nil {
		return 0, nil, err
	}
	return opcode, payload, nil
}
