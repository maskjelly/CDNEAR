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

**Same Wi-Fi** — they use the `192.168.…` line.

**Other Wi-Fi** — `host` opens a public TCP tunnel and prints a second `go run . join …` line. Send that one. No router port-forward. The tunnel is only a pipe to your process; the room and password still live on the host computer.

## What it is

One TCP server, many clients, shared password. Names, join/leave, typing, images (`/img path`), and your draft is not wiped when someone else sends a line. Pictures land in `inbox/`.

If the host has `GEMINI_API_KEY` or a local `gemini.key`, Gemini sits in the room, reads every line, and only speaks when a person would — not on every message. No slash command.

Not encrypted. Anyone with the password who can reach the port is in the room.
