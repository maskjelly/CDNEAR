# cdnear

Direct tunnel chat. One TCP connection between you and your friend. No ntfy, no account, no third-party chat server. Messages only go on that socket.

## Friend: start here

```bash
git clone https://github.com/maskjelly/CDNEAR.git
cd CDNEAR
chmod +x check.sh
./check.sh
```

Need Go 1.21+ from https://go.dev/dl/ (macOS: `brew install go`). Reopen the terminal after installing.

## Chat

**You (host)**
```bash
go run . host
```

It asks `your name:`, then prints the `go run . join …` line for your friend.

**Friend**
```bash
go run . join THE.ADDRESS:9000
```

Use the address `host` printed. Type, press enter. `/quit` leaves.

Same Wi-Fi: use the LAN address (`192.168.…`).

Different houses: the host must forward **TCP 9000** on their router to this computer. Friend joins the public address `host` printed.

## What this is

A TCP tunnel and a terminal chat on top of it. Names, join/leave, typing indicator, and your draft stays if a message arrives mid-type.

Not encrypted. Anyone who can reach the host port can connect. Only run it with someone you trust.
