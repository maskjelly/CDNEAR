package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/term"
)

type lineIO struct {
	mu       sync.Mutex
	buf      []byte
	prompt   string
	status   string
	onChange func(string)
	in       *bufio.Reader
	raw      *term.State
	fd       int
	closed   bool
}

func newLineIO(prompt string) *lineIO {
	l := &lineIO{
		prompt: prompt,
		in:     bufio.NewReader(os.Stdin),
		fd:     int(os.Stdin.Fd()),
	}
	if term.IsTerminal(l.fd) {
		if st, err := term.MakeRaw(l.fd); err == nil {
			l.raw = st
		}
	}
	return l
}

func (l *lineIO) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return
	}
	l.closed = true
	if l.raw != nil {
		_ = term.Restore(l.fd, l.raw)
		l.raw = nil
	}
	fmt.Print("\r\n")
}

func (l *lineIO) Prompt() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		l.drawInputLocked()
	}
}

func (l *lineIO) SetStatus(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed || l.status == s {
		return
	}
	l.status = s
	l.drawInputLocked()
}

func (l *lineIO) Incoming(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return
	}
	fmt.Print("\r\033[2K" + s + "\n")
	l.drawInputLocked()
}

func (l *lineIO) drawInputLocked() {
	fmt.Print("\r\033[2K")
	if l.status != "" {
		fmt.Print(l.status + "  ")
	}
	fmt.Print(l.prompt + string(l.buf))
}

func (l *lineIO) ReadLine() (string, error) {
	if l.raw == nil {
		return l.readCooked()
	}
	for {
		r, _, err := l.in.ReadRune()
		if err != nil {
			return "", err
		}
		switch r {
		case 3:
			return "", io.EOF
		case 4:
			l.mu.Lock()
			empty := len(l.buf) == 0
			l.mu.Unlock()
			if empty {
				return "", io.EOF
			}
		case '\r', '\n':
			l.mu.Lock()
			if l.closed {
				l.mu.Unlock()
				return "", io.EOF
			}
			line := string(l.buf)
			l.buf = l.buf[:0]
			fmt.Print("\r\n")
			l.drawInputLocked()
			l.mu.Unlock()
			l.fire("")
			return line, nil
		case 127, 8:
			l.mu.Lock()
			if len(l.buf) > 0 {
				_, n := utf8.DecodeLastRune(l.buf)
				l.buf = l.buf[:len(l.buf)-n]
				l.drawInputLocked()
			}
			text := string(l.buf)
			l.mu.Unlock()
			l.fire(text)
		case 27:
			drainEscape(l.in)
		default:
			if r == '\t' || unicode.IsPrint(r) {
				l.mu.Lock()
				l.buf = utf8.AppendRune(l.buf, r)
				fmt.Print(string(r))
				text := string(l.buf)
				l.mu.Unlock()
				l.fire(text)
			}
		}
	}
}

func (l *lineIO) readCooked() (string, error) {
	line, err := l.in.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return line, nil
}

func (l *lineIO) fire(text string) {
	if l.onChange != nil {
		l.onChange(text)
	}
}

func drainEscape(in *bufio.Reader) {
	b, err := in.ReadByte()
	if err != nil {
		return
	}
	if b != '[' && b != 'O' {
		return
	}
	for {
		b, err = in.ReadByte()
		if err != nil {
			return
		}
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '~' {
			return
		}
	}
}
