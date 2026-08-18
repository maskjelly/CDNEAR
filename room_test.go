package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestRoomSendAndReceive(t *testing.T) {
	topic := "cdnear-test-" + shortHash(time.Now().String())
	want := "aabbccdd hello-from-test"
	got := make(chan string, 1)

	go func() {
		req, err := http.NewRequest(http.MethodGet, ntfyBase+"/"+topic+"/json", nil)
		if err != nil {
			return
		}
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			var ev struct {
				Event   string `json:"event"`
				Message string `json:"message"`
			}
			if json.Unmarshal(sc.Bytes(), &ev) != nil || ev.Event != "message" {
				continue
			}
			got <- ev.Message
			return
		}
	}()

	time.Sleep(400 * time.Millisecond)
	if err := sendRoom(topic, want); err != nil {
		t.Fatal(err)
	}

	select {
	case msg := <-got:
		if msg != want {
			t.Fatalf("got %q", msg)
		}
	case <-time.After(12 * time.Second):
		t.Fatal("did not see posted message")
	}
}
