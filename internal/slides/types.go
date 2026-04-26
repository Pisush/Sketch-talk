package slides

// Slide represents a single rasterized slide page with extracted text.
type Slide struct {
	Index    int
	ImagePNG []byte
	TextRaw  string
}

// SlideSet holds all slides extracted from a PDF.
type SlideSet struct {
	Slides     []Slide
	TotalPages int
}
