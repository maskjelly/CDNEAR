package main

import (
	"bufio"
	"crypto/subtle"
	"net"
	"sync"
	"time"
)

type hub struct {
	pass  string
	mu    sync.Mutex
	conns map[net.Conn]string
}

func newHub(pass string) *hub {
	return &hub{pass: pass, conns: map[net.Conn]string{}}
}

func (h *hub) checkPass(got string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(h.pass)) == 1
}

func (h *hub) add(c net.Conn, name string) {
	h.mu.Lock()
	h.conns[c] = name
	h.mu.Unlock()
}

func (h *hub) remove(c net.Conn) {
	h.mu.Lock()
	delete(h.conns, c)
	h.mu.Unlock()
}

func (h *hub) broadcast(skip net.Conn, m wire) {
	b, err := encodeWire(m)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.conns {
		if c == skip {
			continue
		}
		_, _ = c.Write(b)
	}
}

func (h *hub) serveConn(conn net.Conn, preauth string) {
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 64*1024), maxWire)
	name := preauth

	if name == "" {
		_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
		if !sc.Scan() {
			return
		}
		m, err := decodeWire(sc.Bytes())
		if err != nil || m.T != "auth" || !h.checkPass(m.Text) {
			b, _ := encodeWire(wire{T: "err", Text: "bad password"})
			_, _ = conn.Write(b)
			return
		}
		name = m.Name
		if name == "" {
			name = "anon"
		}
		b, _ := encodeWire(wire{T: "ok"})
		if _, err := conn.Write(b); err != nil {
			return
		}
		_ = conn.SetDeadline(time.Time{})
	}

	h.add(conn, name)
	h.broadcast(conn, wire{T: "join", Name: name})

	for sc.Scan() {
		m, err := decodeWire(sc.Bytes())
		if err != nil {
			continue
		}
		switch m.T {
		case "msg", "type", "stop", "leave", "img":
			if m.T == "img" && len(m.Data) > maxWire {
				continue
			}
			m.Name = name
			h.broadcast(conn, m)
		}
	}
	h.remove(conn)
	h.broadcast(nil, wire{T: "leave", Name: name})
}

func serveListener(ln net.Listener, h *hub) error {
	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		go h.serveConn(c, "")
	}
}
