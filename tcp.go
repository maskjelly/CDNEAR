package main

import (
	"fmt"
	"net"
)

func host(port string) error {
	name := askName()

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	defer ln.Close()

	fmt.Println()
	fmt.Println("tunnel listening — only you and whoever connects")
	if ips := localIPv4s(); len(ips) > 0 {
		fmt.Println("same network:")
		for _, ip := range ips {
			fmt.Printf("  go run . join %s:%s\n", ip, port)
		}
	}
	if pub := publicTCP(port); pub != "" {
		fmt.Println("other networks (forward TCP " + port + " on your router first):")
		fmt.Printf("  go run . join %s\n", pub)
	}
	fmt.Println("waiting...")

	conn, err := ln.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()
	return tunnelChat(conn, name)
}

func join(addr string) error {
	name := askName()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("could not reach %s: %w", addr, err)
	}
	defer conn.Close()
	return tunnelChat(conn, name)
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
