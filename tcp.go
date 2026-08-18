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

	ln, err := net.Listen("tcp4", ":"+port)
	if err != nil {
		return err
	}
	defer ln.Close()

	h := newHub(pass)
	go serveListener(ln, h)

	fmt.Println()
	fmt.Println("server is accepting joins now")
	if ips := localIPv4s(); len(ips) > 0 {
		fmt.Println("same Wi-Fi:")
		for _, ip := range ips {
			fmt.Printf("  go run . join %s:%s\n", ip, port)
		}
	}

	fmt.Println("opening a public tunnel for other Wi-Fi...")
	tun, err := startPublicTunnel(port)
	if err != nil {
		fmt.Println("public tunnel failed:", err)
		fmt.Println("only same-Wi-Fi joins will work")
	} else {
		defer tun.Close()
		fmt.Println("other Wi-Fi — send this line:")
		fmt.Printf("  go run . join %s\n", tun.Addr)
	}
	fmt.Println("same password for everyone.")

	conn, err := net.DialTimeout("tcp4", net.JoinHostPort("127.0.0.1", port), 3*time.Second)
	if err != nil {
		return fmt.Errorf("could not attach host to the room: %w", err)
	}
	defer conn.Close()
	if err := loginConn(conn, name, pass); err != nil {
		return err
	}
	return tunnelChat(conn, name, false)
}

func join(addr string) error {
	name := askName()
	pass, err := askPassword("password: ")
	if err != nil {
		return err
	}

	addr = normalizeJoinAddr(addr)
	fmt.Fprintf(os.Stderr, "connecting to %s...\n", addr)
	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return fmt.Errorf("could not reach %s — use the 192.168 address the host printed if you are on the same Wi-Fi", addr)
	}
	defer conn.Close()

	fmt.Fprintln(os.Stderr, "logging in...")
	if err := loginConn(conn, name, pass); err != nil {
		return err
	}
	return tunnelChat(conn, name, false)
}

func loginConn(conn net.Conn, name, pass string) error {
	if err := writeWire(conn, wire{T: "auth", Name: name, Text: pass}); err != nil {
		return err
	}
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	br := bufio.NewReader(conn)
	line, err := br.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("no reply from server (is host actually running?): %w", err)
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
	return nil
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
