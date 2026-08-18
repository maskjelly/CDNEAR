package main

import (
	"fmt"
	"os"
)

const defaultPort = "9000"

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "host":
		port := defaultPort
		if len(os.Args) > 2 {
			port = os.Args[2]
		}
		err = host(port)
	case "join":
		if len(os.Args) < 3 {
			err = fmt.Errorf("usage: go run . join host:port")
			break
		}
		err = join(os.Args[2])
	case "meet":
		peer := ""
		if len(os.Args) > 2 {
			peer = os.Args[2]
		}
		err = meet(peer)
	case "relay":
		port := defaultPort
		if len(os.Args) > 2 {
			port = os.Args[2]
		}
		err = relay(port)
	case "via":
		if len(os.Args) < 4 {
			err = fmt.Errorf("usage: go run . via relay-host:port room")
			break
		}
		err = via(os.Args[2], os.Args[3])
	case "check":
		err = check()
	case "help", "-h", "--help":
		usage(os.Stdout)
		return
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage(w *os.File) {
	fmt.Fprint(w, `cdnear — terminal chat (all Go)

First time / is my machine ready:
  ./check.sh
  go run . check

Same Wi-Fi / same network:
  you:     go run . host
  friend:  go run . join 192.168.x.x:9000

Different houses (no server — you swap codes):
  both:    go run . meet
           tell your friend your code, paste theirs

If meet fails (some phones / strict NATs), run a relay
on any machine with a public IP, then both join it:
  server:  go run . relay
  both:    go run . via that.server:9000 secret-room

Type /quit to leave a chat.
`)
}
