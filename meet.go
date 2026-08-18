package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

func meet(peerRaw string) error {
	pc, err := net.ListenUDP("udp", &net.UDPAddr{Port: 0})
	if err != nil {
		return err
	}
	defer pc.Close()

	fmt.Println("asking the internet who you are (STUN)...")
	pub, err := stunPublicAddr(pc)
	if err != nil {
		return fmt.Errorf("could not discover public address: %w", err)
	}

	port := pc.LocalAddr().(*net.UDPAddr).Port
	fmt.Println()
	fmt.Println("give this to your friend:")
	if code := encodeAddrCode(pub); code != "" {
		fmt.Printf("  code    %s\n", code)
	}
	fmt.Printf("  public  %s\n", pub)
	for _, ip := range localIPv4s() {
		local := &net.UDPAddr{IP: net.ParseIP(ip), Port: port}
		fmt.Printf("  local   %s", local)
		if c := encodeAddrCode(local); c != "" {
			fmt.Printf("   (%s)", c)
		}
		fmt.Println()
	}
	fmt.Println()
	fmt.Println("you both need to be in meet at the same time.")

	if peerRaw == "" {
		fmt.Print("paste their code or host:port: ")
		sc := bufio.NewScanner(os.Stdin)
		if !sc.Scan() {
			if err := sc.Err(); err != nil {
				return err
			}
			return fmt.Errorf("no address given")
		}
		peerRaw = sc.Text()
	}

	peer, err := parsePeer(peerRaw)
	if err != nil {
		return fmt.Errorf("bad friend address: %w", err)
	}

	peer, err = punch(pc, peer)
	if err != nil {
		return err
	}
	return udpChat(pc, peer)
}

func punch(pc *net.UDPConn, peer *net.UDPAddr) (*net.UDPAddr, error) {
	fmt.Printf("punching a hole to %s ...\n", peer)
	deadline := time.Now().Add(25 * time.Second)
	buf := make([]byte, 2048)

	for time.Now().Before(deadline) {
		_, _ = pc.WriteToUDP([]byte("C1 PUNCH"), peer)
		_ = pc.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		n, from, err := pc.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return nil, err
		}
		if !from.IP.Equal(peer.IP) {
			continue
		}
		if !strings.HasPrefix(string(buf[:n]), "C1 ") {
			continue
		}
		_ = pc.SetReadDeadline(time.Time{})
		fmt.Println("hole is open")
		return from, nil
	}
	_ = pc.SetReadDeadline(time.Time{})
	return nil, fmt.Errorf("could not reach your friend — their network is probably blocking inbound packets.\ntry the same Wi-Fi (go run . host / join), or run a relay on a public machine")
}

func udpChat(pc *net.UDPConn, peer *net.UDPAddr) error {
	fmt.Printf("-- connected to %s --\n", peer)
	fmt.Println("type a message and press enter. /quit to leave.")

	lio := newLineIO("you> ")
	defer lio.Close()
	lio.Prompt()

	var wmu sync.Mutex
	write := func(b []byte) error {
		wmu.Lock()
		defer wmu.Unlock()
		_, err := pc.WriteToUDP(b, peer)
		return err
	}

	errc := make(chan error, 3)

	go func() {
		buf := make([]byte, 2048)
		seen := map[string]time.Time{}
		for {
			_ = pc.SetReadDeadline(time.Now().Add(30 * time.Second))
			n, from, err := pc.ReadFromUDP(buf)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				errc <- err
				return
			}
			if !from.IP.Equal(peer.IP) {
				continue
			}
			msg := string(buf[:n])
			switch {
			case msg == "C1 PUNCH", msg == "C1 PING":
				continue
			case strings.HasPrefix(msg, "C1 MSG "):
				rest := strings.TrimPrefix(msg, "C1 MSG ")
				id, text, ok := strings.Cut(rest, " ")
				if !ok {
					continue
				}
				if _, dup := seen[id]; dup {
					continue
				}
				seen[id] = time.Now()
				for k, t := range seen {
					if time.Since(t) > time.Minute {
						delete(seen, k)
					}
				}
				lio.Incoming("friend> " + text)
			}
		}
	}()

	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for range t.C {
			if err := write([]byte("C1 PING")); err != nil {
				errc <- err
				return
			}
		}
	}()

	go func() {
		for {
			line, err := lio.ReadLine()
			if err != nil {
				if err == io.EOF {
					errc <- nil
					return
				}
				errc <- err
				return
			}
			if strings.TrimSpace(line) == "/quit" {
				errc <- nil
				return
			}
			if line == "" {
				continue
			}
			id := newMsgID()
			if err := write([]byte("C1 MSG " + id + " " + line)); err != nil {
				errc <- err
				return
			}
		}
	}()

	err := <-errc
	fmt.Println()
	return err
}

func newMsgID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
