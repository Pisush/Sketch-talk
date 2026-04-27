package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/pisush/sketch-talk/internal/ai"
	"github.com/pisush/sketch-talk/internal/demo"
)

func newGenerateDemoCmd() *cobra.Command {
	var topic, outPath string

	cmd := &cobra.Command{
		Use:   "generate-demo",
		Short: "Generate a demo slides PDF using Claude",
		Example: `  sketch-talk generate-demo --topic "Colors of the rainbow" --out ./slides/demo.pdf`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey := os.Getenv("ANTHROPIC_API_KEY")
			if apiKey == "" {
				return fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
			}
			if topic == "" {
				return fmt.Errorf("--topic is required")
			}

			client := ai.NewClient(apiKey)

			log.Printf("Generating slides about %q with Claude…", topic)
			if err := demo.Generate(context.Background(), client, topic, outPath); err != nil {
				return err
			}
			log.Printf("Demo PDF saved to %s", outPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&topic, "topic", "", "Topic for the demo slides")
	cmd.Flags().StringVar(&outPath, "out", "./demo.pdf", "Output PDF path")
	return cmd
}
