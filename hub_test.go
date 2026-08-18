package main

import (
	"bufio"
	"net"
	"testing"
	"time"
)

func startTestHub(t *testing.T, pass string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go serveListener(ln, newHub(pass))
	return ln.Addr().String()
}

func loginTest(t *testing.T, addr, name, pass string) (net.Conn, *bufio.Reader) {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeWire(c, wire{T: "auth", Name: name, Text: pass}); err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(c)
	line, err := br.ReadBytes('\n')
	if err != nil {
		t.Fatal(err)
	}
	m, err := decodeWire(line[:len(line)-1])
	if err != nil {
		t.Fatal(err)
	}
	if m.T != "ok" {
		c.Close()
		t.Fatalf("login: %+v", m)
	}
	return c, br
}

func TestBadPassword(t *testing.T) {
	addr := startTestHub(t, "s3cret")
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := writeWire(c, wire{T: "auth", Name: "eve", Text: "nope"}); err != nil {
		t.Fatal(err)
	}
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	br := bufio.NewReader(c)
	line, err := br.ReadBytes('\n')
	if err != nil {
		t.Fatal(err)
	}
	m, err := decodeWire(line[:len(line)-1])
	if err != nil {
		t.Fatal(err)
	}
	if m.T != "err" {
		t.Fatalf("got %+v", m)
	}
}

func TestPasswordBroadcast(t *testing.T) {
	addr := startTestHub(t, "s3cret")
	a, ar := loginTest(t, addr, "alice", "s3cret")
	defer a.Close()
	b, _ := loginTest(t, addr, "bob", "s3cret")
	defer b.Close()

	_ = a.SetDeadline(time.Now().Add(3 * time.Second))
	line, err := ar.ReadBytes('\n')
	if err != nil {
		t.Fatal(err)
	}
	join, err := decodeWire(line[:len(line)-1])
	if err != nil {
		t.Fatal(err)
	}
	if join.T != "join" || join.Name != "bob" {
		t.Fatalf("got %+v", join)
	}

	if err := writeWire(b, wire{T: "msg", Name: "bob", Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	line, err = ar.ReadBytes('\n')
	if err != nil {
		t.Fatal(err)
	}
	msg, err := decodeWire(line[:len(line)-1])
	if err != nil {
		t.Fatal(err)
	}
	if msg.T != "msg" || msg.Name != "bob" || msg.Text != "hi" {
		t.Fatalf("got %+v", msg)
	}
}
