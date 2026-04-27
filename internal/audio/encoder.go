package audio

import (
	"bytes"
	"encoding/binary"
)

// EncodeWAV encodes PCM int16 samples (mono, 16kHz) into a WAV byte slice.
func EncodeWAV(samples []int16, sampleRate int) []byte {
	numSamples := len(samples)
	dataSize := numSamples * 2 // 16-bit = 2 bytes per sample
	totalSize := 36 + dataSize

	buf := bytes.NewBuffer(make([]byte, 0, totalSize+8))
	le := binary.LittleEndian

	writeStr := func(s string) { buf.WriteString(s) }
	writeU32 := func(v uint32) {
		b := make([]byte, 4)
		le.PutUint32(b, v)
		buf.Write(b)
	}
	writeU16 := func(v uint16) {
		b := make([]byte, 2)
		le.PutUint16(b, v)
		buf.Write(b)
	}

	// RIFF header
	writeStr("RIFF")
	writeU32(uint32(totalSize))
	writeStr("WAVE")

	// fmt chunk
	writeStr("fmt ")
	writeU32(16) // chunk size
	writeU16(1)  // PCM format
	writeU16(1)  // mono
	writeU32(uint32(sampleRate))
	writeU32(uint32(sampleRate * 2)) // byte rate
	writeU16(2)                      // block align
	writeU16(16)                     // bits per sample

	// data chunk
	writeStr("data")
	writeU32(uint32(dataSize))
	for _, s := range samples {
		b := make([]byte, 2)
		le.PutUint16(b, uint16(s))
		buf.Write(b)
	}

	return buf.Bytes()
}
