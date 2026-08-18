package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

// 1x1 PNG
const tinyPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

func TestImageRoundTrip(t *testing.T) {
	raw, err := base64.StdEncoding.DecodeString(tinyPNG)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "dot.png")
	if err := os.WriteFile(src, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	name, b64, got, err := loadImage(src)
	if err != nil {
		t.Fatal(err)
	}
	if name != "dot.png" || len(got) == 0 || b64 == "" {
		t.Fatalf("load: %s %d", name, len(got))
	}

	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	path, saved, err := saveImage(name, b64)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if len(saved) != len(raw) {
		t.Fatalf("saved %d want %d", len(saved), len(raw))
	}
}

func TestRejectNonImage(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := loadImage(src); err == nil {
		t.Fatal("expected error")
	}
}
