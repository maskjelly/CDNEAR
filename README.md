# cdnear

A password-protected chat server you run yourself. Others join with the address and the password. No ntfy, no account, no third-party chat host.

Anyone can host: run this on a computer people can reach, share the address and password.

## Friend / anyone joining

```bash
git clone https://github.com/maskjelly/CDNEAR.git
cd CDNEAR
go run . join HOST:9000
```

It asks for your name and the room password.

## Host

```bash
go run . host
```

It asks for your name and a room password. You stay in the chat. It prints the `join` line for everyone else.

**Same Wi-Fi** — they use the `192.168.…` address it prints.

**From the internet** — the host computer must already be reachable on TCP 9000 (a VPS, a cloud VM, or a home box that already has a public port). This binary will not punch through a home router by itself. If the machine has no public IP, joiners outside your network cannot connect.

## What it is

One TCP server, many clients, shared password. Names, join/leave, typing, and your draft is not wiped when someone else sends a line.

Not encrypted. Anyone with the password who can reach the port is in the room.
