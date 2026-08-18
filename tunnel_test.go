package main

import "testing"

func TestParseTunnelAddr(t *testing.T) {
	in := "You are not authenticated.\ntcp://sgvbx-205-254-163-7.run.pinggy-free.link:43531\n"
	got := parseTunnelAddr(in)
	if got != "sgvbx-205-254-163-7.run.pinggy-free.link:43531" {
		t.Fatalf("got %q", got)
	}
	if parseTunnelAddr("no url here") != "" {
		t.Fatal("expected empty")
	}
}

func TestNormalizeJoinAddr(t *testing.T) {
	if got := normalizeJoinAddr("tcp://host.example:1234"); got != "host.example:1234" {
		t.Fatalf("got %q", got)
	}
}
