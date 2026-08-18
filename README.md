# cdnear

Terminal chat in Go. You and a friend type the same room word and talk. Works across the world.

## Friend: start here

```bash
git clone https://github.com/maskjelly/CDNEAR.git
cd CDNEAR
chmod +x check.sh
./check.sh
```

Need Go 1.21+ from https://go.dev/dl/ (macOS: `brew install go`). Reopen the terminal after installing. `./check.sh` should end with `ready.`

## Chat

Pick any secret word. Both of you run the same command:

```bash
go run . room pineapple-42
```

Type, press enter. `/quit` leaves.

## Other commands

| Command | What it does |
| --- | --- |
| `./check.sh` | Is Go installed, then network checks |
| `go run . check` | Same checks once Go is there |
| `go run . room <word>` | Chat from anywhere |
| `go run . host` | Same Wi-Fi only |
| `go run . join host:port` | Same Wi-Fi only |

Chat is not encrypted. Use a word only you two know.
