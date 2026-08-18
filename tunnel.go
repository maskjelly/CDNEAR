package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os/exec"
	"regexp"
	"strings"
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

var tcpURL = regexp.MustCompile(`tcp://([A-Za-z0-9._-]+):(\d+)`)

func parseTunnelAddr(s string) string {
	m := tcpURL.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return net.JoinHostPort(m[1], m[2])
}

func startPublicTunnel(localPort string) (*publicTunnel, error) {
	if _, err := exec.LookPath("ssh"); err != nil {
		return nil, fmt.Errorf("ssh not found")
	}

	cmd := exec.Command("ssh",
		"-tt",
		"-p", "443",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "GlobalKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=12",
		"-o", "ServerAliveInterval=20",
		"-o", "ServerAliveCountMax=3",
		"-R", "0:127.0.0.1:"+localPort,
		"tcp@a.pinggy.io",
	)
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

	addrCh := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(pr)
		for sc.Scan() {
			if a := parseTunnelAddr(sc.Text()); a != "" {
				select {
				case addrCh <- a:
				default:
				}
			}
		}
	}()

	t := &publicTunnel{cmd: cmd}
	go func() {
		_ = cmd.Wait()
		_ = pw.Close()
		_ = stdin.Close()
	}()

	select {
	case addr := <-addrCh:
		t.Addr = addr
		return t, nil
	case <-time.After(20 * time.Second):
		t.Close()
		return nil, fmt.Errorf("tunnel did not print a public address")
	}
}

func normalizeJoinAddr(addr string) string {
	return strings.TrimPrefix(strings.TrimSpace(addr), "tcp://")
}
