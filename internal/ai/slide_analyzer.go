package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/pisush/sketch-talk/internal/config"
	"github.com/pisush/sketch-talk/internal/slides"
)

const maxSlides = 20

// AnalyzeSlides sends slide images to Claude and returns the initial sketchnote skeleton.
func AnalyzeSlides(ctx context.Context, client anthropic.Client, cfg *config.TalkConfig, ss *slides.SlideSet) (*SketchSkeleton, error) {
	slideSet := ss.Slides
	if len(slideSet) > maxSlides {
		slideSet = slideSet[:maxSlides]
	}

	content := []anthropic.ContentBlockParamUnion{
		anthropic.NewTextBlock(fmt.Sprintf(
			"Talk: %q\nSpeaker: %s (%s)\nConference: %s %s\n\nAnalyze these %d slides and produce the sketchnote layout JSON:",
			cfg.TalkTitle, cfg.SpeakerName, cfg.SpeakerHandle,
			cfg.ConferenceName, cfg.ConferenceTag,
			len(slideSet),
		)),
	}

	for _, slide := range slideSet {
		if len(slide.ImagePNG) == 0 {
			continue
		}
		encoded := base64.StdEncoding.EncodeToString(slide.ImagePNG)
		content = append(content, anthropic.NewImageBlockBase64("image/png", encoded))
	}

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: SystemSlideAnalysis},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(content...),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude slide analysis: %w", err)
	}

	raw := extractText(msg)
	var skeleton SketchSkeleton
	if err := json.Unmarshal([]byte(raw), &skeleton); err != nil {
		return nil, fmt.Errorf("parse skeleton JSON: %w\nraw: %s", err, raw)
	}
	return &skeleton, nil
}

// extractText pulls the text content from the first text block of a message.
func extractText(msg *anthropic.Message) string {
	for _, block := range msg.Content {
		if block.Type == "text" {
			return block.Text
		}
	}
	return ""
}
