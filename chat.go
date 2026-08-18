package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	cReset  = "\033[0m"
	cDim    = "\033[2m"
	cCyan   = "\033[36m"
	cGreen  = "\033[32m"
	cYellow = "\033[33m"
)

func tunnelChat(conn net.Conn, myName string, sendJoin bool) error {
	fmt.Printf("%s-- in the room --%s\n", cGreen, cReset)
	fmt.Println("type and press enter. /img path  to send a picture. /quit to leave.")

	lio := newLineIO(myName + "> ")
	defer lio.Close()

	var wmu sync.Mutex
	send := func(m wire) error {
		b, err := encodeWire(m)
		if err != nil {
			return err
		}
		wmu.Lock()
		defer wmu.Unlock()
		_, err = conn.Write(b)
		return err
	}

	if sendJoin {
		if err := send(wire{T: "join", Name: myName}); err != nil {
			return err
		}
	}

	note := tcpTyper(send, myName)
	lio.onChange = note
	lio.Prompt()

	errc := make(chan error, 2)
	peer := "friend"

	go func() {
		sc := bufio.NewScanner(conn)
		sc.Buffer(make([]byte, 64*1024), maxWire)
		for sc.Scan() {
			m, err := decodeWire(sc.Bytes())
			if err != nil {
				continue
			}
			if m.Name != "" {
				peer = m.Name
			}
			switch m.T {
			case "join":
				lio.Incoming(fmt.Sprintf("%s* %s joined%s", cGreen, peer, cReset))
			case "leave":
				lio.SetStatus("")
				lio.Incoming(fmt.Sprintf("%s* %s left%s", cDim, peer, cReset))
			case "type":
				lio.SetStatus(cYellow + peer + " is typing..." + cReset)
			case "stop":
				lio.SetStatus("")
			case "msg":
				lio.SetStatus("")
				stamp := time.Now().Format("15:04")
				lio.Incoming(fmt.Sprintf("%s%s%s %s%s>%s %s", cDim, stamp, cReset, cCyan, peer, cReset, m.Text))
			case "img":
				lio.SetStatus("")
				path, raw, err := saveImage(m.Text, m.Data)
				if err != nil {
					lio.Incoming(fmt.Sprintf("%s* %s sent an image we could not save: %v%s", cDim, peer, err, cReset))
					break
				}
				lio.ShowImage(fmt.Sprintf("%s%s>%s sent %s → %s", cCyan, peer, cReset, m.Text, path), raw)
			}
		}
		if err := sc.Err(); err != nil {
			errc <- err
			return
		}
		errc <- io.EOF
	}()

	go func() {
		for {
			line, err := lio.ReadLine()
			if err != nil {
				if err == io.EOF {
					_ = send(wire{T: "leave", Name: myName})
					errc <- nil
					return
				}
				errc <- err
				return
			}
			if strings.TrimSpace(line) == "/quit" {
				_ = send(wire{T: "leave", Name: myName})
				errc <- nil
				return
			}
			if strings.TrimSpace(line) == "" {
				continue
			}
			if path, ok := strings.CutPrefix(strings.TrimSpace(line), "/img "); ok {
				note("")
				name, b64, raw, err := loadImage(strings.Trim(path, `"'`))
				if err != nil {
					lio.Incoming(fmt.Sprintf("%s* %v%s", cDim, err, cReset))
					continue
				}
				if err := send(wire{T: "img", Name: myName, Text: name, Data: b64}); err != nil {
					errc <- err
					return
				}
				lio.ShowImage(fmt.Sprintf("%syou sent %s%s", cDim, name, cReset), raw)
				continue
			}
			note("")
			if err := send(wire{T: "msg", Name: myName, Text: line}); err != nil {
				errc <- err
				return
			}
		}
	}()

	err := <-errc
	if err == io.EOF {
		fmt.Println("tunnel closed")
		return nil
	}
	return err
}

func tcpTyper(send func(wire) error, name string) func(string) {
	var mu sync.Mutex
	on := false
	var timer *time.Timer
	stop := func() {
		mu.Lock()
		defer mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		if !on {
			return
		}
		on = false
		go send(wire{T: "stop", Name: name})
	}
	return func(text string) {
		if strings.TrimSpace(text) == "" {
			stop()
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if !on {
			on = true
			go send(wire{T: "type", Name: name})
		}
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(2*time.Second, stop)
	}
}

func askName() string {
	fmt.Print("your name: ")
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return "anon"
	}
	var b strings.Builder
	n := 0
	for _, r := range strings.TrimSpace(sc.Text()) {
		if !unicode.IsPrint(r) {
			continue
		}
		n++
		if n > 20 {
			break
		}
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		return "anon"
	}
	return b.String()
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
