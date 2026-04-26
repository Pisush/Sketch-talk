package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

// ProcessTranscript sends a transcript diff and current canvas state to Claude,
// returning new elements to add to the sketchnote.
func ProcessTranscript(
	ctx context.Context,
	client anthropic.Client,
	transcriptText string,
	canvasPNG []byte,
	placedElements []SketchElement,
) (*TranscriptUpdate, error) {
	placedJSON, _ := json.Marshal(placedElements)

	encoded := base64.StdEncoding.EncodeToString(canvasPNG)

	content := []anthropic.ContentBlockParamUnion{
		anthropic.NewTextBlock(fmt.Sprintf(
			"New transcript segment:\n%s\n\nAlready placed elements:\n%s",
			transcriptText, placedJSON,
		)),
		anthropic.NewImageBlockBase64("image/png", encoded),
	}

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: SystemTranscriptUpdate},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(content...),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude transcript update: %w", err)
	}

	raw := extractText(msg)
	var update TranscriptUpdate
	if err := json.Unmarshal([]byte(raw), &update); err != nil {
		return nil, fmt.Errorf("parse update JSON: %w\nraw: %s", err, raw)
	}
	return &update, nil
}
