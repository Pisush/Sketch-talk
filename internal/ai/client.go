package ai

import (
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// NewClient creates a new Anthropic client using the given API key.
func NewClient(apiKey string) anthropic.Client {
	return anthropic.NewClient(option.WithAPIKey(apiKey))
}
