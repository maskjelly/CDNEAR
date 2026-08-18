package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/term"
)

func host(port string) error {
	name := askName()
	pass, err := askPassword("room password: ")
	if err != nil {
		return err
	}
	if pass == "" {
		return fmt.Errorf("password required")
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	defer ln.Close()

	fmt.Println()
	fmt.Println("server up — anyone with the address and password can join")
	if ips := localIPv4s(); len(ips) > 0 {
		fmt.Println("same network:")
		for _, ip := range ips {
			fmt.Printf("  go run . join %s:%s\n", ip, port)
		}
	}
	if pub := publicTCP(port); pub != "" {
		fmt.Println("if this computer already has a public IP (VPS / port already open):")
		fmt.Printf("  go run . join %s\n", pub)
	}
	fmt.Println("they will be asked for the same password.")

	h := newHub(pass)
	go serveListener(ln, h)

	local, remote := net.Pipe()
	go h.serveConn(remote, name)
	return tunnelChat(local, name, false)
}

func join(addr string) error {
	name := askName()
	pass, err := askPassword("password: ")
	if err != nil {
		return err
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("could not reach %s: %w", addr, err)
	}
	defer conn.Close()

	if err := writeWire(conn, wire{T: "auth", Name: name, Text: pass}); err != nil {
		return err
	}
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	br := bufio.NewReader(conn)
	line, err := br.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("server closed during login: %w", err)
	}
	reply, err := decodeWire(bytes.TrimRight(line, "\r\n"))
	if err != nil {
		return err
	}
	if reply.T != "ok" {
		if reply.Text != "" {
			return fmt.Errorf("%s", reply.Text)
		}
		return fmt.Errorf("login failed")
	}
	_ = conn.SetDeadline(time.Time{})

	return tunnelChat(&prefacedConn{Conn: conn, leftover: br}, name, false)
}

func writeWire(c net.Conn, m wire) error {
	b, err := encodeWire(m)
	if err != nil {
		return err
	}
	_, err = c.Write(b)
	return err
}

func askPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", err
		}
		return "", nil
	}
	return sc.Text(), nil
}

func publicTCP(port string) string {
	pc, err := net.ListenUDP("udp", &net.UDPAddr{Port: 0})
	if err != nil {
		return ""
	}
	defer pc.Close()
	pub, err := stunPublicAddr(pc)
	if err != nil {
		return ""
	}
	return net.JoinHostPort(pub.IP.String(), port)
}
