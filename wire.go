package main

import (
	"bytes"
	"encoding/json"
)

const (
	maxImage = 4 << 20
	maxWire  = 6 << 20
)

type wire struct {
	T    string `json:"t"`
	Name string `json:"n,omitempty"`
	Text string `json:"x,omitempty"`
	Data string `json:"d,omitempty"`
}

func encodeWire(m wire) ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func decodeWire(line []byte) (wire, error) {
	var m wire
	err := json.Unmarshal(bytes.TrimSpace(line), &m)
	return m, err
}
