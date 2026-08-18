package main

import "encoding/json"

type wire struct {
	T    string `json:"t"`
	Name string `json:"n,omitempty"`
	Text string `json:"x,omitempty"`
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
	err := json.Unmarshal(line, &m)
	return m, err
}
