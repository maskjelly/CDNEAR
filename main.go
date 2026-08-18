package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "listen" {
		listen()
		return
	}

	addr := "127.0.0.1:9000"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	send(addr)
}

func listen() {
	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer ln.Close()
	fmt.Println("listening on :9000")

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Printf("got from %s: %s\n", conn.RemoteAddr(), buf[:n])
	_, _ = conn.Write([]byte("got your packet"))
}

func send(addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	// Write bytes onto the TCP connection; the kernel turns them into packets.
	_, err = conn.Write([]byte("HI I AM MR NETWORK MAN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("reply: %s\n", buf[:n])
}
