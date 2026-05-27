package main

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// decodeWAV parses a minimal PCM WAV — the format ffmpeg writes for our packs
// (PCM s16le). It returns interleaved int16 samples, the sample rate, and the
// channel count. It is intentionally small: we only ever feed it our own files.
func decodeWAV(b []byte) (samples []int16, rate int, channels int, err error) {
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, 0, 0, errors.New("not a RIFF/WAVE file")
	}
	var bitsPerSample, dataStart, dataLen int
	for pos := 12; pos+8 <= len(b); {
		id := string(b[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))
		if size < 0 { // 32-bit overflow guard: a huge size must not rewind pos
			return nil, 0, 0, errors.New("invalid chunk size")
		}
		body := pos + 8
		switch id {
		case "fmt ":
			if body+16 > len(b) {
				return nil, 0, 0, errors.New("truncated fmt chunk")
			}
			if audioFmt := binary.LittleEndian.Uint16(b[body : body+2]); audioFmt != 1 {
				return nil, 0, 0, fmt.Errorf("unsupported WAV format %d (need PCM)", audioFmt)
			}
			channels = int(binary.LittleEndian.Uint16(b[body+2 : body+4]))
			rate = int(binary.LittleEndian.Uint32(b[body+4 : body+8]))
			bitsPerSample = int(binary.LittleEndian.Uint16(b[body+14 : body+16]))
		case "data":
			dataStart, dataLen = body, size
		}
		pos = body + size + (size & 1) // chunks are word-aligned
	}
	if bitsPerSample != 16 {
		return nil, 0, 0, fmt.Errorf("unsupported bit depth %d (need 16)", bitsPerSample)
	}
	if rate <= 0 || channels <= 0 {
		return nil, 0, 0, fmt.Errorf("invalid format (rate=%d, channels=%d)", rate, channels)
	}
	if dataStart == 0 {
		return nil, 0, 0, errors.New("no data chunk")
	}
	if dataStart+dataLen > len(b) {
		dataLen = len(b) - dataStart
	}
	n := dataLen / 2
	samples = make([]int16, n)
	for i := 0; i < n; i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(b[dataStart+2*i:]))
	}
	return samples, rate, channels, nil
}
