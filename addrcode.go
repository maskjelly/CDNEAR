package main

import (
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"unicode"
)

func encodeAddrCode(addr *net.UDPAddr) string {
	ip4 := addr.IP.To4()
	if ip4 == nil {
		return ""
	}
	raw := make([]byte, 6)
	copy(raw[:4], ip4)
	binary.BigEndian.PutUint16(raw[4:], uint16(addr.Port))
	s := strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw))
	if len(s) < 10 {
		return s
	}
	return s[:5] + "-" + s[5:10]
}

func parsePeer(s string) (*net.UDPAddr, error) {
	orig := strings.TrimSpace(s)
	if orig == "" {
		return nil, fmt.Errorf("empty address")
	}

	compact := strings.ToUpper(strings.ReplaceAll(orig, "-", ""))
	if len(compact) == 10 && isBase32(compact) {
		raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(compact)
		if err == nil && len(raw) >= 6 {
			ip := net.IPv4(raw[0], raw[1], raw[2], raw[3])
			port := int(binary.BigEndian.Uint16(raw[4:6]))
			return &net.UDPAddr{IP: ip, Port: port}, nil
		}
	}

	return net.ResolveUDPAddr("udp", orig)
}

func isBase32(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) && (r < '2' || r > '7') {
			return false
		}
		if unicode.IsUpper(r) && r > 'Z' {
			return false
		}
	}
	return true
}
