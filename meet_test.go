package main

import (
	"net"
	"testing"
	"time"
)

func TestPunchLocalhost(t *testing.T) {
	a, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	errc := make(chan error, 2)
	go func() {
		_, err := punch(a, b.LocalAddr().(*net.UDPAddr))
		errc <- err
	}()
	go func() {
		_, err := punch(b, a.LocalAddr().(*net.UDPAddr))
		errc <- err
	}()

	timeout := time.After(5 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				t.Fatal(err)
			}
		case <-timeout:
			t.Fatal("timed out punching localhost")
		}
	}
}
