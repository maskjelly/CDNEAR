package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

func TestRelayDeliversMessage(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		rooms := map[string]net.Conn{}
		var mu sync.Mutex
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				_ = pairRelay(c, &mu, rooms)
			}(c)
		}
	}()

	c1, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Close()
	c2, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Close()

	if _, err := fmt.Fprintf(c1, "ROOM testdeliver\n"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if _, err := fmt.Fprintf(c2, "ROOM testdeliver\n"); err != nil {
		t.Fatal(err)
	}

	r1 := bufio.NewReader(c1)
	r2 := bufio.NewReader(c2)
	if line, err := r1.ReadString('\n'); err != nil || line != "WAIT\n" {
		t.Fatalf("c1 first line %q %v", line, err)
	}
	if line, err := r1.ReadString('\n'); err != nil || line != "READY\n" {
		t.Fatalf("c1 second line %q %v", line, err)
	}
	if line, err := r2.ReadString('\n'); err != nil || line != "READY\n" {
		t.Fatalf("c2 first line %q %v", line, err)
	}

	if _, err := fmt.Fprintln(c2, "hello-via-relay"); err != nil {
		t.Fatal(err)
	}

	got, err := r1.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello-via-relay\n" {
		t.Fatalf("got %q", got)
	}
}
