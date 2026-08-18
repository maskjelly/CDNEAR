package main

import (
	"fmt"
	"os"
	"strings"
)

const defaultPort = "9000"

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "room":
		if len(os.Args) < 3 {
			err = fmt.Errorf("usage: go run . room <secret-word>")
			break
		}
		err = room(strings.Join(os.Args[2:], " "))
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

Chat from anywhere (same word, both of you):
  go run . room secret-word

First time:
  ./check.sh

Same Wi-Fi only:
  go run . host
  go run . join 192.168.x.x:9000

Type /quit to leave.
`)
}
