package main

import (
	"bufio"
	"crypto/subtle"
	"net"
	"strings"
	"sync"
	"time"
)

type chatLine struct {
	Name string
	Text string
}

type hub struct {
	pass        string
	mu          sync.Mutex
	conns       map[net.Conn]string
	log         []chatLine
	geminiBusy  bool
	geminiAgain bool
}

func newHub(pass string) *hub {
	return &hub{pass: pass, conns: map[net.Conn]string{}}
}

func (h *hub) note(name, text string) {
	h.mu.Lock()
	h.log = append(h.log, chatLine{Name: name, Text: text})
	if len(h.log) > 80 {
		h.log = h.log[len(h.log)-80:]
	}
	h.mu.Unlock()
}

func (h *hub) history() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var b strings.Builder
	for _, ln := range h.log {
		b.WriteString(ln.Name)
		b.WriteString(": ")
		b.WriteString(ln.Text)
		b.WriteByte('\n')
	}
	return b.String()
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
	targets := make([]net.Conn, 0, len(h.conns))
	for c := range h.conns {
		if c != skip {
			targets = append(targets, c)
		}
	}
	h.mu.Unlock()
	for _, c := range targets {
		_ = c.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_, _ = c.Write(b)
		_ = c.SetWriteDeadline(time.Time{})
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
	h.note("*", name+" joined")
	go h.geminiSee()

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
			switch m.T {
			case "msg":
				h.note(name, m.Text)
				go h.geminiSee()
			case "img":
				h.note(name, "sent an image "+m.Text)
				go h.geminiSee()
			}
		}
	}
	h.remove(conn)
	h.broadcast(nil, wire{T: "leave", Name: name})
	h.note("*", name+" left")
	go h.geminiSee()
}

func (h *hub) seatGemini() {
	if geminiKey() == "" {
		return
	}
	h.note("*", "gemini joined")
	h.broadcast(nil, wire{T: "join", Name: "gemini"})
	h.sayGemini("hey, i'm in the room")
}

func (h *hub) geminiSee() {
	if geminiKey() == "" {
		return
	}
	h.mu.Lock()
	if h.geminiBusy {
		h.geminiAgain = true
		h.mu.Unlock()
		return
	}
	h.geminiBusy = true
	h.mu.Unlock()
	go h.geminiTurn()
}

func (h *hub) geminiTurn() {
	for {
		prompt := geminiPersona + "\n\nTranscript so far:\n" + h.history() + "\nYour next line as gemini:"
		ans, err := askGemini(prompt)
		if err == nil {
			ans = strings.TrimSpace(ans)
			if ans != "" && !strings.EqualFold(ans, "PASS") {
				if len(ans) > 800 {
					ans = ans[:800] + "…"
				}
				h.sayGemini(ans)
			}
		}
		h.mu.Lock()
		if !h.geminiAgain {
			h.geminiBusy = false
			h.mu.Unlock()
			return
		}
		h.geminiAgain = false
		h.mu.Unlock()
	}
}

func (h *hub) sayGemini(text string) {
	h.broadcast(nil, wire{T: "type", Name: "gemini"})
	time.Sleep(350 * time.Millisecond)
	h.broadcast(nil, wire{T: "stop", Name: "gemini"})
	h.broadcast(nil, wire{T: "msg", Name: "gemini", Text: text})
	h.note("gemini", text)
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
