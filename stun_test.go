package main

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestParseXORMappedIPv4(t *testing.T) {
	var txid [12]byte
	copy(txid[:], []byte("123456789012"))

	ip := net.IPv4(203, 0, 113, 9).To4()
	port := uint16(53122)

	attr := make([]byte, 8)
	attr[1] = 0x01
	binary.BigEndian.PutUint16(attr[2:4], port^uint16(stunMagic>>16))
	copy(attr[4:8], ip)
	magic := []byte{0x21, 0x12, 0xA4, 0x42}
	for i := 0; i < 4; i++ {
		attr[4+i] ^= magic[i]
	}

	pkt := make([]byte, 20+4+8)
	binary.BigEndian.PutUint16(pkt[0:2], 0x0101)
	binary.BigEndian.PutUint16(pkt[2:4], 4+8)
	binary.BigEndian.PutUint32(pkt[4:8], stunMagic)
	copy(pkt[8:20], txid[:])
	binary.BigEndian.PutUint16(pkt[20:22], 0x0020)
	binary.BigEndian.PutUint16(pkt[22:24], 8)
	copy(pkt[24:32], attr)

	got, err := parseBindingSuccess(pkt, txid)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IP.Equal(ip) || got.Port != int(port) {
		t.Fatalf("got %s want %s:%d", got, ip, port)
	}
}
