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
	fmt.Fprint(w, `cdnear — password room

  host:   go run . host
  anyone: go run . join <address>

Host sets a password and gets a join line for other Wi-Fi.
Joiners use that line and the same password.

Type /quit to leave.
`)
}
