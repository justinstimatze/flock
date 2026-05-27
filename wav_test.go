package main

import "testing"

// TestDecodeWAVEmbedded decodes a real bundled honk and checks its format.
func TestDecodeWAVEmbedded(t *testing.T) {
	b, err := honksFS.ReadFile("honks/goose/honk00.wav")
	if err != nil {
		t.Fatalf("read embedded honk: %v", err)
	}
	samples, rate, channels, err := decodeWAV(b)
	if err != nil {
		t.Fatalf("decodeWAV: %v", err)
	}
	if rate != 44100 {
		t.Errorf("rate = %d, want 44100", rate)
	}
	if channels != 1 {
		t.Errorf("channels = %d, want 1", channels)
	}
	if len(samples) == 0 {
		t.Error("decoded zero samples")
	}
}

// TestDecodeWAVRejectsBadInput makes sure malformed data is rejected rather than
// panicking — this is the path a custom-pack directory could hit.
func TestDecodeWAVRejectsBadInput(t *testing.T) {
	cases := map[string][]byte{
		"empty":         {},
		"not riff":      []byte("XXXXxxxxWAVEfmt "),
		"riff not wave": []byte("RIFF\x10\x00\x00\x00XXXXfmt "),
		"no data chunk": append([]byte("RIFF\x24\x00\x00\x00WAVEfmt \x10\x00\x00\x00"),
			// PCM, 1ch, 44100Hz, 16-bit, but no data chunk follows
			0x01, 0x00, 0x01, 0x00, 0x44, 0xac, 0x00, 0x00,
			0x88, 0x58, 0x01, 0x00, 0x02, 0x00, 0x10, 0x00),
	}
	for name, b := range cases {
		if _, _, _, err := decodeWAV(b); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}
