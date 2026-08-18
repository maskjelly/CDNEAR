# cdnear

Terminal chat in Go. Two people run the same program and type to each other.

This is the first slice of a lightweight tunnel: send a packet to another machine, get something back, then talk.

## Friend: start here

Do these in order. If a step fails, stop and fix it before chatting.

### 1. Get the folder

You need this project on disk (`main.go`, `go.mod`, `check.sh`). `cd` into it:

```bash
cd CDNEAR
```

### 2. Is everything installed?

**macOS / Linux** — one command checks Go, files, ports, DNS, and internet:

```bash
chmod +x check.sh
./check.sh
```

**Windows (PowerShell)** — there is no bash script. Run the two commands below.

**If `./check.sh` is not an option**, check by hand:

```bash
go version
```

You want a line like `go version go1.24.4 darwin/arm64`.

| Result | What to do |
| --- | --- |
| `go version go1.21` or newer | Good. Continue. |
| `command not found` / `'go' is not recognized` | Install Go, **close and reopen the terminal**, run `go version` again. |
| Go older than 1.21 | Install a current Go from the link below. |

Install Go: [https://go.dev/dl/](https://go.dev/dl/)

macOS with Homebrew:

```bash
brew install go
```

Then, still in this folder:

```bash
go run . check
```

That second command only works if Go is installed. It checks:

- Go actually runs
- you are in this project folder
- this machine can open TCP (same-Wi-Fi chat) and UDP (internet `meet`)
- port `9000` is free (or tells you to pick another)
- you have a LAN address for same-Wi-Fi chat
- DNS works
- outbound internet works
- STUN can see your public address (`meet` needs this)

You want it to end with `ready.`

```
  ok    go runtime     go1.24.4 darwin/arm64
  ok    project files  go.mod and main.go found
  ok    tcp bind       can open a TCP socket
  ok    tcp :9000      free (default host/relay port)
  ok    udp bind       can open a UDP socket (needed for meet)
  ok    local ipv4     192.168.x.x
  ok    dns            stun.l.google.com resolves
  ok    outbound tcp   can make outbound TCP connections
  ok    stun           your.public.ip:port
ready.
```

`FAIL` means that piece cannot run until you fix it. `warn` means one mode may not work (for example no STUN → skip `meet`, use same Wi-Fi or a relay).

### 3. Chat

**Same Wi-Fi** (most reliable):

```bash
# you
go run . host

# friend — use the address host printed
go run . join 192.168.x.x:9000
```

**Different houses**, both at the same time:

```bash
go run . meet
```

Swap the codes it prints, paste the other person's code, press enter.

If `meet` cannot punch through (some phone hotspots / strict NATs), someone with a public IP runs `go run . relay` and you both run:

```bash
go run . via that.server:9000 secret-room
```

Type a line and press enter. `/quit` leaves.

## Commands

| Command | What it does |
| --- | --- |
| `./check.sh` | Preflight: is Go installed, then runs `go run . check` |
| `go run . check` | Network / port / STUN readiness (needs Go) |
| `go run . host [port]` | Wait for one friend on this network |
| `go run . join host:port` | Connect to a host |
| `go run . meet` | Internet chat; swap codes |
| `go run . relay [port]` | Pair two friends (run on a public machine) |
| `go run . via host:port room` | Join a relay room |
| `go run . help` | Print usage |

Default port is `9000`.

## What must be true

| Need | Why | How we check |
| --- | --- | --- |
| Go 1.21+ on `PATH` | The program is Go. Nothing runs without it. | `go version` / `./check.sh` |
| This folder (`go.mod`, `main.go`) | `go run .` builds from these files | `./check.sh` and `go run . check` |
| A terminal | You type messages here | you are reading this in one |
| Same Wi-Fi **or** a path to the other person | Packets have to arrive | `check` prints which modes work |
| UDP + STUN for `meet` | Finds your public address and opens a hole | `go run . check` STUN line |
| Both people in `meet` at the same time | NAT mappings die if you wait | run it together |

Chat is not encrypted. Only use it with someone you already trust.

## If a check fails

**`go` not found**  
Install from [https://go.dev/dl/](https://go.dev/dl/). Reopen the terminal. On macOS, `brew install go` also works. Confirm with `go version`.

**`go run .` errors about `package` or files**  
You are not in this folder. `cd` to the directory that contains `main.go`.

**`tcp :9000` in use**  
Something else is on 9000. Host on another port: `go run . host 9001` and join that port.

**`local ipv4` missing**  
You are not on a LAN (or only have IPv6). Same-Wi-Fi `host` / `join` will not work. Use `meet` or a relay.

**`stun` / `dns` / `outbound tcp` fail**  
No working internet, or a firewall is blocking it. Same-Wi-Fi chat can still work. `meet` and `via` will not.

**`meet` says it could not reach your friend**  
Both of you must be in `meet` at the same time. If it still fails, you are likely behind a strict NAT — use the same Wi-Fi or a relay.

## Objectives

- Lightweight Tailscale-style tunnel (long term)
- Right now: a packet to another machine, a reply, and a live terminal chat
