package demo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pisush/sketch-talk/assets"
	"golang.org/x/image/font"
)

// slide holds the content for one generated slide.
type slide struct {
	Title   string   `json:"title"`
	Bullets []string `json:"bullets"`
}

// Generate asks Claude to invent slides about topic, renders them to PNGs,
// and writes a multi-page PDF to outputPath.
func Generate(ctx context.Context, client anthropic.Client, topic, outputPath string) error {
	slides, err := generateSlideContent(ctx, client, topic)
	if err != nil {
		return fmt.Errorf("generate slide content: %w", err)
	}

	boldFont, regularFont, err := loadFonts()
	if err != nil {
		return fmt.Errorf("load fonts: %w", err)
	}

	pngBuffers := make([][]byte, len(slides))
	for i, s := range slides {
		buf, err := renderSlide(s, boldFont, regularFont)
		if err != nil {
			return fmt.Errorf("render slide %d: %w", i, err)
		}
		pngBuffers[i] = buf
	}

	paths, cleanup, err := writeTempPNGs(pngBuffers)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := api.ImportImagesFile(paths, outputPath, nil, nil); err != nil {
		return fmt.Errorf("create pdf: %w", err)
	}
	return nil
}

func generateSlideContent(ctx context.Context, client anthropic.Client, topic string) ([]slide, error) {
	prompt := fmt.Sprintf(`Generate exactly 6 conference talk slides about "%s".
Return a JSON array only — no prose, no markdown, no code fences.
Each element: {"title": "...", "bullets": ["...", "...", "..."]}
3–4 bullets per slide. Make the content factual and interesting.`, topic)

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, err
	}

	raw := ""
	for _, block := range msg.Content {
		if block.Type == "text" {
			raw = block.Text
			break
		}
	}

	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		if idx := strings.Index(raw, "\n"); idx >= 0 {
			raw = raw[idx+1:]
		}
		raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")
		raw = strings.TrimSpace(raw)
	}

	var slides []slide
	if err := json.Unmarshal([]byte(raw), &slides); err != nil {
		return nil, fmt.Errorf("parse slides JSON: %w\nraw: %s", err, raw)
	}
	return slides, nil
}

func renderSlide(s slide, boldFont, regularFont *truetype.Font) ([]byte, error) {
	const w, h = 1024, 768
	dc := gg.NewContext(w, h)

	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Title bar.
	dc.SetRGB(0.15, 0.35, 0.65)
	dc.DrawRectangle(0, 0, w, 120)
	dc.Fill()

	titleFace := truetype.NewFace(boldFont, &truetype.Options{Size: 42, DPI: 96, Hinting: font.HintingFull})
	dc.SetFontFace(titleFace)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringWrapped(s.Title, w/2, 60, 0.5, 0.5, float64(w)-60, 1.2, gg.AlignCenter)

	bulletFace := truetype.NewFace(regularFont, &truetype.Options{Size: 28, DPI: 96, Hinting: font.HintingFull})
	dc.SetFontFace(bulletFace)
	dc.SetRGB(0.1, 0.1, 0.1)

	y := 180.0
	for _, bullet := range s.Bullets {
		dc.DrawStringWrapped("•  "+bullet, 80, y, 0, 0, float64(w)-160, 1.3, gg.AlignLeft)
		y += 100
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func loadFonts() (bold, regular *truetype.Font, err error) {
	boldData, err := assets.FontFS.ReadFile("fonts/Caveat-Bold.ttf")
	if err != nil {
		return nil, nil, err
	}
	bold, err = truetype.Parse(boldData)
	if err != nil {
		return nil, nil, err
	}
	regularData, err := assets.FontFS.ReadFile("fonts/Caveat-Regular.ttf")
	if err != nil {
		return nil, nil, err
	}
	regular, err = truetype.Parse(regularData)
	return bold, regular, err
}

func writeTempPNGs(pngs [][]byte) (paths []string, cleanup func(), err error) {
	dir, err := os.MkdirTemp("", "sketch-demo-*")
	if err != nil {
		return nil, nil, err
	}
	paths = make([]string, len(pngs))
	for i, data := range pngs {
		p := fmt.Sprintf("%s/slide-%02d.png", dir, i+1)
		if err := os.WriteFile(p, data, 0644); err != nil {
			os.RemoveAll(dir)
			return nil, nil, err
		}
		paths[i] = p
	}
	return paths, func() { os.RemoveAll(dir) }, nil
}
