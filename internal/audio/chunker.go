package audio

import (
	"context"
	"time"
)

// AudioChunk is a WAV-encoded audio chunk ready for transcription.
type AudioChunk struct {
	WAVBytes  []byte
	StartTime time.Time
	Duration  time.Duration
}

// ChunkSamples accumulates PCM samples and emits fixed-size chunks with overlap.
func ChunkSamples(ctx context.Context, sampleCh <-chan []int16, chunkCh chan<- AudioChunk, chunkSeconds, overlapSeconds int) {
	chunkSize := chunkSeconds * SampleRate
	overlapSize := overlapSeconds * SampleRate
	buf := make([]int16, 0, chunkSize*2)
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			// Flush remaining samples as a final partial chunk.
			if len(buf) > SampleRate { // at least 1 second
				chunkCh <- AudioChunk{
					WAVBytes:  EncodeWAV(buf, SampleRate),
					StartTime: startTime,
					Duration:  time.Duration(len(buf)) * time.Second / SampleRate,
				}
			}
			return
		case samples, ok := <-sampleCh:
			if !ok {
				return
			}
			buf = append(buf, samples...)
			if len(buf) >= chunkSize {
				chunk := make([]int16, chunkSize)
				copy(chunk, buf[:chunkSize])
				chunkCh <- AudioChunk{
					WAVBytes:  EncodeWAV(chunk, SampleRate),
					StartTime: startTime,
					Duration:  time.Duration(chunkSeconds) * time.Second,
				}
				// Retain overlap tail for next chunk.
				if overlapSize < len(buf) {
					buf = buf[chunkSize-overlapSize:]
				} else {
					buf = buf[:0]
				}
				startTime = time.Now()
			}
		}
	}
}
