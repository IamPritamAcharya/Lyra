package fingerprint

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// ReadCanonicalWAV reads little-endian PCM16 mono WAV at landmark-v1's rate.
func ReadCanonicalWAV(path string) ([]int16, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open wav: %w", err)
	}
	defer f.Close()
	var riff [12]byte
	if _, err := io.ReadFull(f, riff[:]); err != nil {
		return nil, fmt.Errorf("read wav header: %w", err)
	}
	if string(riff[:4]) != "RIFF" || string(riff[8:]) != "WAVE" {
		return nil, fmt.Errorf("invalid wav container")
	}
	var channels, rate, bits uint32
	var data []byte
	for {
		var chunk [8]byte
		if _, err := io.ReadFull(f, chunk[:]); err != nil {
			break
		}
		size := binary.LittleEndian.Uint32(chunk[4:])
		buf := make([]byte, size)
		if _, err := io.ReadFull(f, buf); err != nil {
			return nil, fmt.Errorf("read wav chunk: %w", err)
		}
		switch string(chunk[:4]) {
		case "fmt ":
			if len(buf) < 16 || binary.LittleEndian.Uint16(buf) != 1 {
				return nil, fmt.Errorf("unsupported wav encoding")
			}
			channels, rate, bits = uint32(binary.LittleEndian.Uint16(buf[2:])), binary.LittleEndian.Uint32(buf[4:]), uint32(binary.LittleEndian.Uint16(buf[14:]))
		case "data":
			data = buf
		}
		if size%2 == 1 {
			if _, err := f.Seek(1, io.SeekCurrent); err != nil {
				return nil, err
			}
		}
	}
	if channels != 1 || rate != SampleRate || bits != 16 || len(data) == 0 || len(data)%2 != 0 {
		return nil, fmt.Errorf("want PCM16 mono %dHz WAV", SampleRate)
	}
	samples := make([]int16, len(data)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(data[i*2:]))
	}
	return samples, nil
}
