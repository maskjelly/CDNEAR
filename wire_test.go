package main

import (
	"bytes"
	"testing"
)

func TestWireRoundTrip(t *testing.T) {
	in := wire{T: "msg", Name: "alex", Text: "hello there"}
	b, err := encodeWire(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(b, []byte("\n")) {
		t.Fatal("expected a newline")
	}
	out, err := decodeWire(bytes.TrimRight(b, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("got %+v want %+v", out, in)
	}
}
