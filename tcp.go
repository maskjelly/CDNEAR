package main

import (
	"fmt"
	"net"
)

func host(port string) error {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	defer ln.Close()

	fmt.Printf("waiting for a friend on port %s\n", port)
	if ips := localIPv4s(); len(ips) > 0 {
		fmt.Println("on the same Wi-Fi they should run:")
		for _, ip := range ips {
			fmt.Printf("  go run . join %s:%s\n", ip, port)
		}
	}

	conn, err := ln.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()
	return chat(conn, "friend")
}

func join(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("could not reach %s: %w", addr, err)
	}
	defer conn.Close()
	return chat(conn, "friend")
}
