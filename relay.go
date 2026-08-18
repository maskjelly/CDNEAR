package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
)

func relay(port string) error {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	defer ln.Close()

	fmt.Printf("relay listening on :%s\n", port)
	fmt.Println("friends join with:")
	fmt.Printf("  go run . via <this-machine>:%s <room>\n", port)

	var mu sync.Mutex
	rooms := map[string]net.Conn{}

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go func(c net.Conn) {
			if err := pairRelay(c, &mu, rooms); err != nil {
				fmt.Println(err)
			}
		}(conn)
	}
}

func pairRelay(c net.Conn, mu *sync.Mutex, rooms map[string]net.Conn) error {
	br := bufio.NewReader(c)
	line, err := br.ReadString('\n')
	if err != nil {
		c.Close()
		return err
	}
	fields := strings.Fields(line)
	if len(fields) != 2 || fields[0] != "ROOM" {
		fmt.Fprintln(c, "ERR use: ROOM <name>")
		c.Close()
		return fmt.Errorf("%s sent a bad handshake", c.RemoteAddr())
	}
	room := fields[1]

	mu.Lock()
	peer, ok := rooms[room]
	if !ok {
		rooms[room] = c
		mu.Unlock()
		fmt.Fprintf(c, "WAIT\n")
		fmt.Printf("room %q waiting for a friend (%s)\n", room, c.RemoteAddr())
		return nil
	}
	delete(rooms, room)
	mu.Unlock()

	if _, err := fmt.Fprintf(peer, "READY\n"); err != nil {
		fmt.Fprintln(c, "ERR peer gone")
		c.Close()
		peer.Close()
		return err
	}
	if _, err := fmt.Fprintf(c, "READY\n"); err != nil {
		c.Close()
		peer.Close()
		return err
	}
	fmt.Printf("room %q connected %s <-> %s\n", room, peer.RemoteAddr(), c.RemoteAddr())

	go func() {
		_, _ = io.Copy(peer, br)
		peer.Close()
		c.Close()
	}()
	_, _ = io.Copy(c, peer)
	c.Close()
	peer.Close()
	return nil
}

func via(addr, room string) error {
	name := askName()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("could not reach relay %s: %w", addr, err)
	}
	defer conn.Close()

	if _, err := fmt.Fprintf(conn, "ROOM %s\n", room); err != nil {
		return err
	}

	br := bufio.NewReader(conn)
	line, err := br.ReadString('\n')
	if err != nil {
		return fmt.Errorf("relay closed the connection: %w", err)
	}
	switch strings.TrimSpace(line) {
	case "WAIT":
		fmt.Printf("waiting in room %q — your friend should run:\n  go run . via %s %s\n", room, addr, room)
		line, err = br.ReadString('\n')
		if err != nil {
			return fmt.Errorf("relay closed while waiting: %w", err)
		}
		if strings.TrimSpace(line) != "READY" {
			return fmt.Errorf("relay said %q", strings.TrimSpace(line))
		}
	case "READY":
	default:
		return fmt.Errorf("relay said %q", strings.TrimSpace(line))
	}

	return tunnelChat(&prefacedConn{Conn: conn, leftover: br}, name, true)
}

// leftover handshake bytes, then the socket
type prefacedConn struct {
	net.Conn
	leftover *bufio.Reader
}

func (c *prefacedConn) Read(p []byte) (int, error) {
	return c.leftover.Read(p)
}
