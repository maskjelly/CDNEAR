package main

import (
	"net"
	"testing"
)

func TestAddrCodeRoundTrip(t *testing.T) {
	in := &net.UDPAddr{IP: net.IPv4(203, 0, 113, 9), Port: 53122}
	code := encodeAddrCode(in)
	if code == "" {
		t.Fatal("expected a code")
	}
	out, err := parsePeer(code)
	if err != nil {
		t.Fatal(err)
	}
	if !out.IP.Equal(in.IP) || out.Port != in.Port {
		t.Fatalf("got %s want %s", out, in)
	}
}

func TestParseHostPort(t *testing.T) {
	out, err := parsePeer("127.0.0.1:9000")
	if err != nil {
		t.Fatal(err)
	}
	if out.Port != 9000 || !out.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("got %s", out)
	}
}
