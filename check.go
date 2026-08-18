package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"time"
)

func check() error {
	fmt.Println("cdnear readiness")
	fmt.Println()

	failed := 0
	ok := func(name, detail string) {
		fmt.Printf("  ok    %-14s %s\n", name, detail)
	}
	warn := func(name, detail string) {
		fmt.Printf("  warn  %-14s %s\n", name, detail)
	}
	fail := func(name, detail string) {
		fmt.Printf("  FAIL  %-14s %s\n", name, detail)
		failed++
	}

	ok("go runtime", fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH))

	if _, err := os.Stat("go.mod"); err != nil {
		fail("project files", "go.mod missing — cd into the CDNEAR folder")
	} else if _, err := os.Stat("main.go"); err != nil {
		fail("project files", "main.go missing — cd into the CDNEAR folder")
	} else {
		ok("project files", "go.mod and main.go found")
	}

	if ln, err := net.Listen("tcp", "127.0.0.1:0"); err != nil {
		fail("tcp bind", err.Error())
	} else {
		ok("tcp bind", "can open a TCP socket")
		ln.Close()
	}

	if ln, err := net.Listen("tcp", ":"+defaultPort); err != nil {
		warn("tcp :"+defaultPort, "in use — host with another port, e.g. go run . host 9001")
	} else {
		ok("tcp :"+defaultPort, "free (default host/relay port)")
		ln.Close()
	}

	pc, err := net.ListenUDP("udp", &net.UDPAddr{Port: 0})
	if err != nil {
		fail("udp bind", err.Error())
	} else {
		ok("udp bind", "can open a UDP socket (needed for meet)")
	}

	ips := localIPv4s()
	if len(ips) == 0 {
		warn("local ipv4", "no LAN address — same-Wi-Fi host/join will not work")
	} else {
		ok("local ipv4", ips[0]+"  (friend on this Wi-Fi uses this)")
		for _, ip := range ips[1:] {
			fmt.Printf("                  %s\n", ip)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	_, dnsErr := net.DefaultResolver.LookupHost(ctx, "stun.l.google.com")
	cancel()
	if dnsErr != nil {
		fail("dns", dnsErr.Error())
	} else {
		ok("dns", "stun.l.google.com resolves")
	}

	dialer := net.Dialer{Timeout: 4 * time.Second}
	if c, err := dialer.Dial("tcp", "1.1.1.1:443"); err != nil {
		warn("outbound tcp", "cannot reach the internet on TCP — via/relay will fail: "+err.Error())
	} else {
		c.Close()
		ok("outbound tcp", "can make outbound TCP connections")
	}

	meetOK := false
	if pc != nil {
		pub, err := stunPublicAddr(pc)
		pc.Close()
		if err != nil {
			warn("stun", "no public address — meet will fail: "+err.Error())
		} else {
			ok("stun", pub.String()+"  (your public address for meet)")
			meetOK = true
		}
	}

	fmt.Println()
	fmt.Println("what you can run")
	if len(ips) > 0 {
		fmt.Printf("  same Wi-Fi     yes    go run . host\n")
		fmt.Printf("                        friend: go run . join %s:%s\n", ips[0], defaultPort)
	} else {
		fmt.Printf("  same Wi-Fi     no     no local IPv4 address\n")
	}
	if meetOK {
		fmt.Printf("  internet meet  yes    go run . meet   (both of you, at the same time)\n")
	} else {
		fmt.Printf("  internet meet  no     STUN failed; try a relay or the same Wi-Fi\n")
	}
	fmt.Printf("  relay join     yes    go run . via <public-server>:%s <room>\n", defaultPort)
	fmt.Println()

	if failed > 0 {
		return fmt.Errorf("%d check(s) failed — fix those before chatting", failed)
	}
	fmt.Println("ready.")
	return nil
}
