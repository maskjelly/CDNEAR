package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type publicTunnel struct {
	Addr string
	cmd  *exec.Cmd
}

func (t *publicTunnel) Close() {
	if t == nil || t.cmd == nil || t.cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-t.cmd.Process.Pid, syscall.SIGTERM)
	_ = t.cmd.Process.Kill()
}

var tcpURL = regexp.MustCompile(`tcp://([A-Za-z0-9._-]+)\s*:\s*(\d+)`)

func parseTunnelAddr(s string) string {
	m := tcpURL.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return net.JoinHostPort(m[1], m[2])
}

func sshTunnelCmd(localPort, server string) *exec.Cmd {
	args := []string{
		"-F", "/dev/null",
		"-p", "443",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "GlobalKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=12",
		"-o", "ServerAliveInterval=20",
		"-o", "ServerAliveCountMax=3",
		"-R", "0:127.0.0.1:" + localPort,
		server,
	}
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return exec.Command("ssh", args...)
	}
	if _, err := exec.LookPath("script"); err == nil {
		switch runtime.GOOS {
		case "darwin":
			return exec.Command("script", append([]string{"-q", "/dev/null", sshPath}, args...)...)
		case "linux":
			var b strings.Builder
			b.WriteString(strconv.Quote(sshPath))
			for _, a := range args {
				b.WriteByte(' ')
				b.WriteString(strconv.Quote(a))
			}
			return exec.Command("script", "-q", "-c", b.String(), "/dev/null")
		}
	}
	return exec.Command(sshPath, append([]string{"-tt"}, args...)...)
}

func startPublicTunnel(localPort string) (*publicTunnel, error) {
	if _, err := exec.LookPath("ssh"); err != nil {
		return nil, fmt.Errorf("ssh not found")
	}
	var last error
	for _, server := range []string{"tcp@a.pinggy.io", "tcp@free.pinggy.io"} {
		tun, err := startPublicTunnelTo(localPort, server)
		if err == nil {
			return tun, nil
		}
		last = err
	}
	return nil, last
}

func startPublicTunnelTo(localPort, server string) (*publicTunnel, error) {
	cmd := sshTunnelCmd(localPort, server)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var mu sync.Mutex
	var acc strings.Builder
	addrCh := make(chan string, 1)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := pr.Read(buf)
			if n > 0 {
				mu.Lock()
				acc.Write(buf[:n])
				blob := acc.String()
				mu.Unlock()
				if a := parseTunnelAddr(blob); a != "" {
					select {
					case addrCh <- a:
					default:
					}
				}
			}
			if err != nil {
				return
			}
		}
	}()

	done := make(chan error, 1)
	t := &publicTunnel{cmd: cmd}
	go func() {
		err := cmd.Wait()
		_ = pw.Close()
		_ = stdin.Close()
		done <- err
	}()

	select {
	case addr := <-addrCh:
		t.Addr = addr
		return t, nil
	case err := <-done:
		mu.Lock()
		out := tail(acc.String(), 600)
		mu.Unlock()
		if err == nil {
			err = fmt.Errorf("ssh exited")
		}
		return nil, fmt.Errorf("%w\n%s", err, out)
	case <-time.After(25 * time.Second):
		t.Close()
		mu.Lock()
		out := tail(acc.String(), 600)
		mu.Unlock()
		return nil, fmt.Errorf("tunnel did not print a public address\n%s", out)
	}
}

func tail(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func normalizeJoinAddr(addr string) string {
	return strings.TrimPrefix(strings.TrimSpace(addr), "tcp://")
}
