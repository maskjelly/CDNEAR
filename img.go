package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

func loadImage(path string) (name, b64 string, raw []byte, err error) {
	raw, err = os.ReadFile(path)
	if err != nil {
		return "", "", nil, err
	}
	if len(raw) > maxImage {
		return "", "", nil, fmt.Errorf("image too big (max %d MB)", maxImage>>20)
	}
	ext, ok := imageExt(raw)
	if !ok {
		return "", "", nil, fmt.Errorf("not a png, jpeg, gif, or webp")
	}
	name = safeName(filepath.Base(path))
	if filepath.Ext(name) == "" {
		name += ext
	}
	return name, base64.StdEncoding.EncodeToString(raw), raw, nil
}

func saveImage(name, b64 string) (string, []byte, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", nil, err
	}
	if len(raw) > maxImage {
		return "", nil, fmt.Errorf("image too big")
	}
	if _, ok := imageExt(raw); !ok {
		return "", nil, fmt.Errorf("not an image")
	}
	if err := os.MkdirAll("inbox", 0o755); err != nil {
		return "", nil, err
	}
	name = safeName(name)
	if name == "" {
		name = "image"
	}
	out := filepath.Join("inbox", time.Now().Format("150405")+"-"+name)
	if err := os.WriteFile(out, raw, 0o644); err != nil {
		return "", nil, err
	}
	return out, raw, nil
}

func imageExt(b []byte) (string, bool) {
	switch {
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return ".png", true
	case bytes.HasPrefix(b, []byte("\xff\xd8\xff")):
		return ".jpg", true
	case bytes.HasPrefix(b, []byte("GIF87a")), bytes.HasPrefix(b, []byte("GIF89a")):
		return ".gif", true
	case len(b) >= 12 && string(b[8:12]) == "WEBP":
		return ".webp", true
	default:
		return "", false
	}
}

func safeName(name string) string {
	name = filepath.Base(name)
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func writeInlineImage(raw []byte) {
	b64 := base64.StdEncoding.EncodeToString(raw)
	switch {
	case os.Getenv("ITERM_SESSION_ID") != "" || os.Getenv("TERM_PROGRAM") == "iTerm.app":
		fmt.Printf("\033]1337;File=inline=1;width=40%%:%s\007\n", b64)
	case os.Getenv("KITTY_WINDOW_ID") != "" || strings.Contains(os.Getenv("TERM"), "kitty"):
		fmt.Printf("\033_Ga=T,f=100,m=0;%s\033\\\n", b64)
	}
}
