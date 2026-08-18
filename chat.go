package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

func chat(conn net.Conn, peerName string) error {
	fmt.Printf("-- connected to %s --\n", peerName)
	fmt.Println("type a message and press enter. /quit to leave.")
	fmt.Print("you> ")

	errc := make(chan error, 2)

	go func() {
		sc := bufio.NewScanner(conn)
		for sc.Scan() {
			line := sc.Text()
			fmt.Printf("\r\033[K%s> %s\nyou> ", peerName, line)
		}
		if err := sc.Err(); err != nil {
			errc <- err
			return
		}
		errc <- io.EOF
	}()

	go func() {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			line := sc.Text()
			if strings.TrimSpace(line) == "/quit" {
				errc <- nil
				return
			}
			if _, err := fmt.Fprintln(conn, line); err != nil {
				errc <- err
				return
			}
			fmt.Print("you> ")
		}
		if err := sc.Err(); err != nil {
			errc <- err
			return
		}
		errc <- nil
	}()

	err := <-errc
	fmt.Println()
	if err == io.EOF {
		fmt.Println("friend disconnected")
		return nil
	}
	return err
}

func localIPv4s() []string {
	var out []string
	ifaces, err := net.InterfaceAddrs()
	if err != nil {
		return out
	}
	for _, a := range ifaces {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			out = append(out, ip4.String())
		}
	}
	return out
}
