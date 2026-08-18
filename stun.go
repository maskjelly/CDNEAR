package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

const stunMagic = 0x2112A442

func stunPublicAddr(pc *net.UDPConn) (*net.UDPAddr, error) {
	servers := []string{
		"stun.l.google.com:19302",
		"stun1.l.google.com:19302",
		"stun.cloudflare.com:3478",
	}
	var last error
	for _, s := range servers {
		addr, err := stunQuery(pc, s)
		if err == nil {
			return addr, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("no stun servers")
	}
	return nil, last
}

func stunQuery(pc *net.UDPConn, server string) (*net.UDPAddr, error) {
	raddr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return nil, err
	}

	var txid [12]byte
	if _, err := rand.Read(txid[:]); err != nil {
		return nil, err
	}
	req := buildBindingRequest(txid)

	buf := make([]byte, 1500)
	var last error
	for try := 0; try < 3; try++ {
		_ = pc.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, err := pc.WriteToUDP(req, raddr); err != nil {
			last = err
			continue
		}
		n, _, err := pc.ReadFromUDP(buf)
		if err != nil {
			last = err
			continue
		}
		addr, err := parseBindingSuccess(buf[:n], txid)
		if err != nil {
			last = err
			continue
		}
		_ = pc.SetReadDeadline(time.Time{})
		return addr, nil
	}
	_ = pc.SetReadDeadline(time.Time{})
	return nil, fmt.Errorf("stun %s: %w", server, last)
}

func buildBindingRequest(txid [12]byte) []byte {
	b := make([]byte, 20)
	binary.BigEndian.PutUint16(b[0:2], 0x0001) // Binding Request
	binary.BigEndian.PutUint16(b[2:4], 0)
	binary.BigEndian.PutUint32(b[4:8], stunMagic)
	copy(b[8:20], txid[:])
	return b
}

func parseBindingSuccess(b []byte, txid [12]byte) (*net.UDPAddr, error) {
	if len(b) < 20 {
		return nil, fmt.Errorf("short stun packet")
	}
	if binary.BigEndian.Uint16(b[0:2]) != 0x0101 {
		return nil, fmt.Errorf("not a binding success")
	}
	if binary.BigEndian.Uint32(b[4:8]) != stunMagic {
		return nil, fmt.Errorf("bad stun magic")
	}
	if string(b[8:20]) != string(txid[:]) {
		return nil, fmt.Errorf("stun transaction mismatch")
	}
	length := int(binary.BigEndian.Uint16(b[2:4]))
	if len(b) < 20+length {
		return nil, fmt.Errorf("truncated stun attributes")
	}

	attrs := b[20 : 20+length]
	var mapped *net.UDPAddr
	for off := 0; off+4 <= len(attrs); {
		typ := binary.BigEndian.Uint16(attrs[off : off+2])
		alen := int(binary.BigEndian.Uint16(attrs[off+2 : off+4]))
		off += 4
		if off+alen > len(attrs) {
			break
		}
		val := attrs[off : off+alen]
		switch typ {
		case 0x0020: // XOR-MAPPED-ADDRESS
			if a := parseSTUNAddr(val, true, b[4:20]); a != nil {
				return a, nil
			}
		case 0x0001: // MAPPED-ADDRESS
			mapped = parseSTUNAddr(val, false, nil)
		}
		off += alen + (4-alen%4)%4
	}
	if mapped != nil {
		return mapped, nil
	}
	return nil, fmt.Errorf("stun response had no mapped address")
}

func parseSTUNAddr(val []byte, xor bool, mask []byte) *net.UDPAddr {
	if len(val) < 4 {
		return nil
	}
	family := val[1]
	port := binary.BigEndian.Uint16(val[2:4])
	if xor {
		port ^= uint16(stunMagic >> 16)
	}
	switch family {
	case 0x01:
		if len(val) < 8 {
			return nil
		}
		ip := make(net.IP, 4)
		copy(ip, val[4:8])
		if xor {
			for i := 0; i < 4; i++ {
				ip[i] ^= mask[i]
			}
		}
		return &net.UDPAddr{IP: ip, Port: int(port)}
	case 0x02:
		if len(val) < 20 {
			return nil
		}
		ip := make(net.IP, 16)
		copy(ip, val[4:20])
		if xor {
			for i := 0; i < 16; i++ {
				ip[i] ^= mask[i]
			}
		}
		return &net.UDPAddr{IP: ip, Port: int(port)}
	default:
		return nil
	}
}
