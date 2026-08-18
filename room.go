package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	ntfyBase = "https://ntfy.sh"
	tagJoin  = "\x01join"
	tagHere  = "\x01here"
	tagLeave = "\x01leave"
)

func room(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("usage: go run . room <secret-word>")
	}

	topic := "cdnear-" + shortHash(name)
	me := newMsgID()

	fmt.Printf("room %q — both of you run:\n  go run . room %s\n", name, name)
	fmt.Println("type a message and press enter. /quit to leave.")
	fmt.Print("you> ")

	errc := make(chan error, 2)
	go func() { errc <- listenRoom(topic, me) }()
	time.Sleep(400 * time.Millisecond)
	_ = sendRoom(topic, me+" "+tagJoin)

	go func() {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			line := sc.Text()
			if strings.TrimSpace(line) == "/quit" {
				_ = sendRoom(topic, me+" "+tagLeave)
				errc <- nil
				return
			}
			if strings.TrimSpace(line) == "" {
				fmt.Print("you> ")
				continue
			}
			if err := sendRoom(topic, me+" "+line); err != nil {
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
	return <-errc
}

func listenRoom(topic, me string) error {
	client := &http.Client{}
	for {
		req, err := http.NewRequest(http.MethodGet, ntfyBase+"/"+topic+"/json", nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "cdnear")
		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			var ev struct {
				Event   string `json:"event"`
				Message string `json:"message"`
			}
			if json.Unmarshal(sc.Bytes(), &ev) != nil || ev.Event != "message" {
				continue
			}
			id, rest, ok := strings.Cut(ev.Message, " ")
			if !ok || id == me {
				continue
			}
			switch rest {
			case tagJoin:
				fmt.Printf("\r\033[K* friend joined\nyou> ")
				_ = sendRoom(topic, me+" "+tagHere)
			case tagHere:
				fmt.Printf("\r\033[K* friend is here\nyou> ")
			case tagLeave:
				fmt.Printf("\r\033[K* friend left\nyou> ")
			default:
				fmt.Printf("\r\033[Kfriend> %s\nyou> ", rest)
			}
		}
		resp.Body.Close()
		time.Sleep(time.Second)
	}
}

func sendRoom(topic, body string) error {
	req, err := http.NewRequest(http.MethodPost, ntfyBase+"/"+topic, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "cdnear")
	req.Header.Set("Priority", "min")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("send failed: %s", resp.Status)
	}
	return nil
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}
