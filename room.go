package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	ntfyBase = "https://ntfy.sh"
	tagJoin  = "\x01join"
	tagHere  = "\x01here"
	tagLeave = "\x01leave"
	tagType  = "\x01type"
	tagStop  = "\x01stop"
	tagMsg   = "\x01msg"
)

func room(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("usage: go run . room <secret-word>")
	}

	who := askName()
	topic := "cdnear-" + shortHash(name)
	me := newMsgID()

	fmt.Printf("room %q — both of you run:\n  go run . room %s\n", name, name)
	fmt.Println("type a message and press enter. /quit to leave.")

	lio := newLineIO(who + "> ")
	defer lio.Close()
	note := newTyper(topic, me, who)
	lio.onChange = note
	lio.Prompt()

	errc := make(chan error, 2)
	go func() { errc <- listenRoom(topic, me, who, lio) }()
	time.Sleep(400 * time.Millisecond)
	_ = sendRoom(topic, pack(me, tagJoin, who, ""))

	go func() {
		for {
			line, err := lio.ReadLine()
			if err != nil {
				if err != io.EOF {
					errc <- err
					return
				}
				_ = sendRoom(topic, pack(me, tagLeave, who, ""))
				errc <- nil
				return
			}
			if strings.TrimSpace(line) == "/quit" {
				_ = sendRoom(topic, pack(me, tagLeave, who, ""))
				errc <- nil
				return
			}
			if strings.TrimSpace(line) == "" {
				continue
			}
			note("")
			if err := sendRoom(topic, pack(me, tagMsg, who, line)); err != nil {
				errc <- err
				return
			}
		}
	}()
	return <-errc
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
		if r == 0x01 || !unicode.IsPrint(r) {
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

func pack(id, tag, user, text string) string {
	s := id + " " + tag
	if user != "" {
		s += " " + user
	}
	if text != "" {
		s += "\x01" + text
	}
	return s
}

func listenRoom(topic, me, myName string, lio *lineIO) error {
	st := &roomState{names: map[string]string{}, typers: map[string]time.Time{}}
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for range t.C {
			st.expire(lio)
		}
	}()

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
			tag, payload, _ := strings.Cut(rest, " ")
			user, text, _ := strings.Cut(payload, "\x01")
			if user == "" {
				user = st.nameOf(id)
			} else {
				st.remember(id, user)
			}
			switch tag {
			case tagJoin:
				st.setTyping(id, false)
				lio.Incoming("* " + user + " joined")
				_ = sendRoom(topic, pack(me, tagHere, myName, ""))
				lio.SetStatus(st.status())
			case tagHere:
				lio.Incoming("* " + user + " is here")
			case tagLeave:
				st.setTyping(id, false)
				lio.Incoming("* " + user + " left")
				lio.SetStatus(st.status())
			case tagType:
				st.remember(id, user)
				st.setTyping(id, true)
				lio.SetStatus(st.status())
			case tagStop:
				st.setTyping(id, false)
				lio.SetStatus(st.status())
			case tagMsg:
				st.setTyping(id, false)
				lio.Incoming(user + "> " + text)
				lio.SetStatus(st.status())
			default:
				// old clients sent "id text"
				st.setTyping(id, false)
				lio.Incoming(user + "> " + rest)
				lio.SetStatus(st.status())
			}
		}
		resp.Body.Close()
		time.Sleep(time.Second)
	}
}

type roomState struct {
	mu     sync.Mutex
	names  map[string]string
	typers map[string]time.Time
}

func (s *roomState) remember(id, name string) {
	if name == "" {
		return
	}
	s.mu.Lock()
	s.names[id] = name
	s.mu.Unlock()
}

func (s *roomState) nameOf(id string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n := s.names[id]; n != "" {
		return n
	}
	return "friend"
}

func (s *roomState) setTyping(id string, on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if on {
		s.typers[id] = time.Now()
	} else {
		delete(s.typers, id)
	}
}

func (s *roomState) expire(lio *lineIO) {
	s.mu.Lock()
	now := time.Now()
	changed := false
	for id, t := range s.typers {
		if now.Sub(t) > 4*time.Second {
			delete(s.typers, id)
			changed = true
		}
	}
	stat := s.statusLocked()
	s.mu.Unlock()
	if changed {
		lio.SetStatus(stat)
	}
}

func (s *roomState) status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusLocked()
}

func (s *roomState) statusLocked() string {
	if len(s.typers) == 0 {
		return ""
	}
	var names []string
	for id := range s.typers {
		n := s.names[id]
		if n == "" {
			n = "friend"
		}
		names = append(names, n)
	}
	if len(names) == 1 {
		return names[0] + " is typing..."
	}
	return strings.Join(names, ", ") + " are typing..."
}

func newTyper(topic, me, name string) func(string) {
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
		go sendRoom(topic, pack(me, tagStop, name, ""))
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
			go sendRoom(topic, pack(me, tagType, name, ""))
		}
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(2*time.Second, stop)
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
