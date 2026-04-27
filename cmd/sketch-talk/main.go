package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/pisush/sketch-talk/internal/config"
	"github.com/pisush/sketch-talk/internal/pipeline"
)

func main() {
	cfg := config.Default()

	root := &cobra.Command{
		Use:   "sketch-talk",
		Short: "Live conference sketchnote generator",
	}

	run := &cobra.Command{
		Use:   "run",
		Short: "Start a live sketchnote session",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
			cfg.OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")

			if cfg.AnthropicAPIKey == "" {
				return fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
			}
			if cfg.OpenAIAPIKey == "" && !cfg.NoAudio {
				return fmt.Errorf("OPENAI_API_KEY environment variable is required (or use --no-audio)")
			}

			orch, err := pipeline.New(cfg)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sig
				cancel()
			}()

			if err := orch.Run(ctx); err != nil {
				return err
			}

			<-ctx.Done()
			orch.Shutdown(cfg.OutputPath, cfg.QuestionsPath)
			return nil
		},
	}

	f := run.Flags()
	f.StringVar(&cfg.TalkTitle, "title", "", "Talk title")
	f.StringVar(&cfg.SpeakerName, "speaker", "", "Speaker name")
	f.StringVar(&cfg.SpeakerHandle, "handle", "", "Speaker social handle (e.g. @alice)")
	f.StringVar(&cfg.ConferenceName, "conference", "", "Conference name")
	f.StringVar(&cfg.ConferenceTag, "hashtag", "", "Conference hashtag (e.g. #gophercon2025)")
	f.StringVar(&cfg.SlidesPath, "slides", "", "Path to slides PDF")
	f.StringVar(&cfg.OutputPath, "output", cfg.OutputPath, "Output file path (.png)")
	f.StringVar(&cfg.QuestionsPath, "questions-output", cfg.QuestionsPath, "Path to save audience questions")
	f.StringVar(&cfg.ListenAddr, "port", cfg.ListenAddr, "HTTP listen address")
	f.IntVar(&cfg.AudioDeviceIndex, "audio-device", cfg.AudioDeviceIndex, "Audio device index (-1 = default)")
	f.IntVar(&cfg.ChunkSeconds, "chunk-seconds", cfg.ChunkSeconds, "Audio chunk length in seconds")
	f.BoolVar(&cfg.NoAudio, "no-audio", false, "Disable audio capture (display-only mode)")

	root.AddCommand(run)
	root.AddCommand(newGenerateDemoCmd())

	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}
