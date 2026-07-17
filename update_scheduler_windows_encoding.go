//go:build windows

package main

import (
	"encoding/binary"
	"unicode/utf16"
)

func encodeUTF16LE(value string) []byte {
	encoded := utf16.Encode([]rune(value))
	data := make([]byte, 2+len(encoded)*2)
	data[0] = 0xff
	data[1] = 0xfe
	for index, item := range encoded {
		binary.LittleEndian.PutUint16(data[2+index*2:], item)
	}
	return data
}
