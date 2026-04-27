package pipeline

import (
	"context"
	"fmt"
	"image"
	"log"
	"strings"
	"time"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/pisush/sketch-talk/internal/ai"
	"github.com/pisush/sketch-talk/internal/audio"
	"github.com/pisush/sketch-talk/internal/config"
	"github.com/pisush/sketch-talk/internal/questions"
	"github.com/pisush/sketch-talk/internal/renderer"
	"github.com/pisush/sketch-talk/internal/server"
	"github.com/pisush/sketch-talk/internal/slides"
	"github.com/pisush/sketch-talk/internal/transcription"
)

// Orchestrator wires all subsystems together.
type Orchestrator struct {
	cfg       *config.TalkConfig
	claude    anthropicsdk.Client
	whisper   *transcription.Client
	canvas    *renderer.Canvas
	animator  *renderer.Animator
	hub       *server.Hub
	srv       *server.Server
	frameCh   chan []byte
	primary   renderer.Color
	liveSeq   int // counter for live_NNN element IDs
	questions *questions.Store
}

// New creates and returns a new Orchestrator.
func New(cfg *config.TalkConfig) (*Orchestrator, error) {
	canvas, err := renderer.NewCanvas()
	if err != nil {
		return nil, fmt.Errorf("create canvas: %w", err)
	}

	qs := questions.NewStore()
	hub := server.NewHub()
	srv := server.NewServer(hub, qs)
	frameCh := make(chan []byte, 32)

	animator := renderer.NewAnimator(canvas, frameCh)

	return &Orchestrator{
		cfg:       cfg,
		claude:    ai.NewClient(cfg.AnthropicAPIKey),
		whisper:   transcription.NewClient(cfg.OpenAIAPIKey),
		canvas:    canvas,
		animator:  animator,
		hub:       hub,
		srv:       srv,
		frameCh:   frameCh,
		primary:   renderer.PrimaryColorFor("blue"),
		questions: qs,
	}, nil
}

// Run starts all subsystems and blocks until ctx is cancelled.
func (o *Orchestrator) Run(ctx context.Context) error {
	// Start WebSocket hub.
	go o.hub.Run()

	// Start HTTP server.
	go func() {
		if err := o.srv.ListenAndServe(o.cfg.ListenAddr); err != nil {
			log.Printf("server: %v", err)
		}
	}()

	// Start frame broadcaster (frameCh → hub + snapshot).
	go o.broadcastFrames(ctx)

	// Push initial blank canvas.
	o.pushSnapshot()

	// Pre-process slides if provided.
	if o.cfg.SlidesPath != "" {
		log.Println("Extracting slides…")
		ss, err := slides.Extract(ctx, o.cfg.SlidesPath)
		if err != nil {
			log.Printf("slide extraction: %v (continuing without slides)", err)
		} else {
			log.Printf("Analyzing %d slides with Claude…", len(ss.Slides))
			skeleton, err := ai.AnalyzeSlides(ctx, o.claude, o.cfg, ss)
			if err != nil {
				log.Printf("slide analysis: %v (continuing without skeleton)", err)
			} else {
				o.primary = renderer.PrimaryColorFor(skeleton.PrimaryColor)
				log.Printf("Theme: %s | %d elements", skeleton.TalkTheme, len(skeleton.Elements))
				o.applySkeleton(ctx, skeleton)
			}
		}
	} else {
		// No slides: draw metadata header/footer directly.
		o.drawMetadata(ctx)
	}

	// Start audio pipeline.
	if err := o.runAudioPipeline(ctx); err != nil {
		log.Printf("audio pipeline: %v", err)
	}

	return nil
}

// applySkeleton animates all skeleton elements onto the canvas.
func (o *Orchestrator) applySkeleton(ctx context.Context, skeleton *ai.SketchSkeleton) {
	// Ensure metadata elements are present.
	skeleton.Elements = prependMetadata(skeleton.Elements, o.cfg)

	for _, el := range skeleton.Elements {
		select {
		case <-ctx.Done():
			return
		default:
		}
		o.enqueueElement(el)
	}
}

// drawMetadata adds the header/footer elements when no slides are given.
func (o *Orchestrator) drawMetadata(ctx context.Context) {
	els := metadataElements(o.cfg)
	for _, el := range els {
		select {
		case <-ctx.Done():
			return
		default:
		}
		o.enqueueElement(el)
	}
}

// enqueueElement resolves an element's bounding box and queues its animation.
func (o *Orchestrator) enqueueElement(el ai.SketchElement) {
	bbox, err := o.canvas.ResolveElement(el)
	if err != nil {
		log.Printf("resolve %s: %v", el.ID, err)
		return
	}

	job := renderer.AnimJob{
		Element: el,
		BBox:    bbox,
		Primary: o.primary,
	}

	// For arrows, look up source and target bounding boxes.
	if el.Kind == ai.KindArrow {
		if from, ok := o.canvas.BBoxFor(el.FromID); ok {
			job.FromBBox = from
		}
		if to, ok := o.canvas.BBoxFor(el.ToID); ok {
			job.ToBBox = to
		}
		if job.FromBBox == (image.Rectangle{}) || job.ToBBox == (image.Rectangle{}) {
			return // skip arrow if endpoints not found
		}
	}

	o.animator.Enqueue(job)
}

// runAudioPipeline starts audio capture, chunking, transcription, and Claude processing.
func (o *Orchestrator) runAudioPipeline(ctx context.Context) error {
	if o.cfg.NoAudio {
		log.Println("Audio disabled (--no-audio); skipping mic capture and transcription.")
		<-ctx.Done()
		return nil
	}

	sampleCh := make(chan []int16, 128)
	chunkCh := make(chan audio.AudioChunk, 4)
	transcriptCh := make(chan transcription.TranscriptSegment, 4)

	// Audio capture.
	go func() {
		if err := audio.StartCapture(ctx, o.cfg.AudioDeviceIndex, sampleCh); err != nil {
			log.Printf("audio capture: %v", err)
		}
	}()

	// Chunker.
	go func() {
		audio.ChunkSamples(ctx, sampleCh, chunkCh, o.cfg.ChunkSeconds, o.cfg.OverlapSeconds)
	}()

	// Transcription worker (sequential).
	go func() {
		var prevText string
		for chunk := range chunkCh {
			seg, err := o.whisper.Transcribe(ctx, chunk, prevText)
			if err != nil {
				log.Printf("transcribe: %v", err)
				continue
			}
			log.Printf("Transcript: %s", seg.Text)
			prevText = seg.Text
			transcriptCh <- seg
		}
		close(transcriptCh)
	}()

	// Claude transcript processor (every 2 segments ≈ 50s).
	go func() {
		var buf strings.Builder
		segCount := 0
		for seg := range transcriptCh {
			buf.WriteString(seg.Text)
			buf.WriteString(" ")
			segCount++
			if segCount < 2 {
				continue
			}
			text := strings.TrimSpace(buf.String())
			buf.Reset()
			segCount = 0

			canvasPNG := o.canvas.Snapshot()
			placed := o.canvas.CommittedIDs()

			update, err := ai.ProcessTranscript(ctx, o.claude, text, canvasPNG, placed)
			if err != nil {
				log.Printf("transcript update: %v", err)
				continue
			}
			for _, el := range update.AddElements {
				o.enqueueElement(el)
			}
		}
	}()

	<-ctx.Done()
	return nil
}

// broadcastFrames fans out frames from the animator to the WebSocket hub.
func (o *Orchestrator) broadcastFrames(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-o.frameCh:
			if !ok {
				return
			}
			o.srv.UpdateSnapshot(frame)
			o.hub.Broadcast <- frame
		}
	}
}

// pushSnapshot broadcasts the current canvas without animation.
func (o *Orchestrator) pushSnapshot() {
	png := o.canvas.Snapshot()
	o.srv.UpdateSnapshot(png)
	select {
	case o.hub.Broadcast <- png:
	default:
	}
}

// Shutdown stops the animator and exports the final sketchnote and questions.
func (o *Orchestrator) Shutdown(outputPath, questionsPath string) {
	log.Println("Shutting down…")
	o.animator.Stop()

	time.Sleep(300 * time.Millisecond)

	png := o.canvas.Snapshot()
	if err := savePNG(png, outputPath); err != nil {
		log.Printf("save sketchnote: %v", err)
	} else {
		log.Printf("Sketchnote saved to %s", outputPath)
	}

	if questionsPath != "" && o.questions.Count() > 0 {
		if err := o.questions.Export(questionsPath); err != nil {
			log.Printf("save questions: %v", err)
		} else {
			log.Printf("%d question(s) saved to %s", o.questions.Count(), questionsPath)
		}
	}
}

// prependMetadata ensures header/footer metadata elements are in the skeleton.
func prependMetadata(elements []ai.SketchElement, cfg *config.TalkConfig) []ai.SketchElement {
	// Check if header banner already present.
	for _, el := range elements {
		if el.Zone == ai.ZoneHeader && el.Kind == ai.KindBanner {
			return append(metadataFooter(cfg), elements...)
		}
	}
	return append(metadataElements(cfg), elements...)
}

func metadataElements(cfg *config.TalkConfig) []ai.SketchElement {
	return append(metadataHeader(cfg), metadataFooter(cfg)...)
}

func metadataHeader(cfg *config.TalkConfig) []ai.SketchElement {
	return []ai.SketchElement{
		{
			ID:       "meta_title",
			Kind:     ai.KindBanner,
			Zone:     ai.ZoneHeader,
			RelX:     0.0, RelY: 0.0, W: 0.65, H: 1.0,
			Text:     cfg.TalkTitle,
			Emphasis: 3,
		},
		{
			ID:       "meta_speaker",
			Kind:     ai.KindBanner,
			Zone:     ai.ZoneHeader,
			RelX:     0.67, RelY: 0.0, W: 0.33, H: 1.0,
			Text:     cfg.SpeakerName + " " + cfg.SpeakerHandle,
			Emphasis: 2,
		},
	}
}

func metadataFooter(cfg *config.TalkConfig) []ai.SketchElement {
	return []ai.SketchElement{
		{
			ID:       "meta_conf",
			Kind:     ai.KindBanner,
			Zone:     ai.ZoneFooter,
			RelX:     0.0, RelY: 0.0, W: 0.5, H: 1.0,
			Text:     cfg.ConferenceName,
			Emphasis: 2,
		},
		{
			ID:       "meta_tag",
			Kind:     ai.KindBanner,
			Zone:     ai.ZoneFooter,
			RelX:     0.5, RelY: 0.0, W: 0.5, H: 1.0,
			Text:     cfg.ConferenceTag,
			Emphasis: 2,
		},
	}
}
