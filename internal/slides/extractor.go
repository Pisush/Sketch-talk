package slides

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Extract rasterizes all pages of a PDF and extracts per-page text.
// It shells out to pdftoppm for rendering; falls back to pdfcpu image extraction.
func Extract(ctx context.Context, pdfPath string) (*SlideSet, error) {
	f, err := os.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	conf := model.NewDefaultConfiguration()
	pageCount, err := api.PageCount(f, conf)
	if err != nil {
		return nil, fmt.Errorf("page count: %w", err)
	}

	slides := make([]Slide, 0, pageCount)

	for i := 1; i <= pageCount; i++ {
		slide := Slide{Index: i}
		slide.ImagePNG, _ = renderPage(ctx, pdfPath, i)
		slide.TextRaw, _ = extractPageText(pdfPath, i)
		slides = append(slides, slide)
	}

	return &SlideSet{Slides: slides, TotalPages: pageCount}, nil
}

// renderPage produces a PNG for one page using pdftoppm, falling back to pdfcpu.
func renderPage(ctx context.Context, pdfPath string, pageNum int) ([]byte, error) {
	_, err := exec.LookPath("pdftoppm")
	if err == nil {
		return renderWithPdftoppm(ctx, pdfPath, pageNum)
	}
	return renderWithPdfcpu(pdfPath, pageNum)
}

func renderWithPdftoppm(ctx context.Context, pdfPath string, pageNum int) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "sketchnote-slides-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	outPrefix := filepath.Join(tmpDir, "slide")
	cmd := exec.CommandContext(ctx, "pdftoppm",
		"-png", "-r", "150",
		"-f", fmt.Sprintf("%d", pageNum),
		"-l", fmt.Sprintf("%d", pageNum),
		pdfPath, outPrefix,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("pdftoppm: %s: %w", out, err)
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil || len(entries) == 0 {
		return nil, fmt.Errorf("pdftoppm produced no output")
	}
	return os.ReadFile(filepath.Join(tmpDir, entries[0].Name()))
}

func renderWithPdfcpu(pdfPath string, pageNum int) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "sketchnote-img-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	pages := []string{fmt.Sprintf("%d", pageNum)}
	conf := model.NewDefaultConfiguration()
	if err := api.ExtractImagesFile(pdfPath, tmpDir, pages, conf); err != nil {
		return nil, fmt.Errorf("pdfcpu extract images: %w", err)
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil || len(entries) == 0 {
		return nil, fmt.Errorf("no images extracted from page %d", pageNum)
	}
	return os.ReadFile(filepath.Join(tmpDir, entries[0].Name()))
}

// extractPageText pulls raw text from a single PDF page via pdfcpu content streams.
func extractPageText(pdfPath string, pageNum int) (string, error) {
	tmpDir, err := os.MkdirTemp("", "sketchnote-text-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	pages := []string{fmt.Sprintf("%d", pageNum)}
	conf := model.NewDefaultConfiguration()
	if err := api.ExtractContentFile(pdfPath, tmpDir, pages, conf); err != nil {
		return "", fmt.Errorf("pdfcpu extract content: %w", err)
	}

	var sb strings.Builder
	entries, _ := os.ReadDir(tmpDir)
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(tmpDir, e.Name()))
		if err != nil {
			continue
		}
		sb.Write(stripPDFOperators(data))
		sb.WriteByte('\n')
	}
	return sb.String(), nil
}

// stripPDFOperators extracts text strings from a PDF content stream (BT...ET blocks).
func stripPDFOperators(content []byte) []byte {
	var out bytes.Buffer
	inText := false
	tokens := bytes.Fields(content)
	for _, tok := range tokens {
		s := string(tok)
		switch s {
		case "BT":
			inText = true
		case "ET":
			inText = false
		default:
			if inText && len(s) > 2 && s[0] == '(' {
				// Strip surrounding parens from PDF text literal
				out.WriteString(s[1:len(s)-1])
				out.WriteByte(' ')
			}
		}
	}
	return out.Bytes()
}
