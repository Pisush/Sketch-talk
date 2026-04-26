package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/pisush/sketch-talk/internal/audio"
)

// TranscriptSegment is a transcribed chunk of speech.
type TranscriptSegment struct {
	Text      string
	StartTime time.Time
}

// Client transcribes audio via the OpenAI Whisper API.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a Whisper transcription client.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Transcribe sends a WAV chunk to Whisper and returns the transcript text.
// prevText is passed as a prompt for context continuity (last 224 chars).
func (c *Client) Transcribe(ctx context.Context, chunk audio.AudioChunk, prevText string) (TranscriptSegment, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	fw, err := mw.CreateFormFile("file", "audio.wav")
	if err != nil {
		return TranscriptSegment{}, err
	}
	if _, err := fw.Write(chunk.WAVBytes); err != nil {
		return TranscriptSegment{}, err
	}

	_ = mw.WriteField("model", "whisper-1")
	_ = mw.WriteField("language", "en")
	_ = mw.WriteField("response_format", "json")

	if len(prevText) > 224 {
		prevText = prevText[len(prevText)-224:]
	}
	if prevText != "" {
		_ = mw.WriteField("prompt", prevText)
	}
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/audio/transcriptions", &body)
	if err != nil {
		return TranscriptSegment{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return TranscriptSegment{}, fmt.Errorf("whisper request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return TranscriptSegment{}, fmt.Errorf("whisper HTTP %d: %s", resp.StatusCode, respBytes)
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return TranscriptSegment{}, fmt.Errorf("whisper parse: %w", err)
	}

	return TranscriptSegment{
		Text:      result.Text,
		StartTime: chunk.StartTime,
	}, nil
}
