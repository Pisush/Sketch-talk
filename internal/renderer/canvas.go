package renderer

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"sync"

	"github.com/fogleman/gg"
	"github.com/pisush/sketch-talk/internal/ai"
)

// DrawnElement pairs a committed SketchElement with its rendered bounding box.
type DrawnElement struct {
	Spec     ai.SketchElement
	BBox     image.Rectangle
	Primary  Color
	FromBBox image.Rectangle // for arrows
	ToBBox   image.Rectangle // for arrows
}

// Canvas is the shared drawing surface and committed element store.
type Canvas struct {
	mu        sync.Mutex
	ctx       *gg.Context
	zones     ZoneMap
	committed []DrawnElement
	// index for arrow resolution: element ID → bounding box
	bboxByID  map[string]image.Rectangle
}

// NewCanvas initialises the gg drawing context and loads fonts.
func NewCanvas() (*Canvas, error) {
	if err := loadFonts(); err != nil {
		return nil, fmt.Errorf("load fonts: %w", err)
	}
	ctx := gg.NewContext(CanvasW, CanvasH)
	c := &Canvas{
		ctx:      ctx,
		zones:    DefaultZoneMap(),
		bboxByID: make(map[string]image.Rectangle),
	}
	drawBackground(ctx)
	return c, nil
}

// Snapshot returns the current canvas as a PNG byte slice (thread-safe).
func (c *Canvas) Snapshot() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.encodePNG()
}

// BBoxFor returns the bounding box of a previously committed element by ID.
func (c *Canvas) BBoxFor(id string) (image.Rectangle, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	bbox, ok := c.bboxByID[id]
	return bbox, ok
}

// CommittedIDs returns a snapshot of all committed element IDs and their zones.
func (c *Canvas) CommittedIDs() []ai.SketchElement {
	c.mu.Lock()
	defer c.mu.Unlock()
	els := make([]ai.SketchElement, len(c.committed))
	for i, d := range c.committed {
		els[i] = d.Spec
	}
	return els
}

// ResolveElement computes the pixel bounding box for an element.
func (c *Canvas) ResolveElement(el ai.SketchElement) (image.Rectangle, error) {
	return c.zones.Resolve(el)
}

// encodePNG encodes the current gg context image to PNG bytes.
// Must be called with c.mu held.
func (c *Canvas) encodePNG() []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, c.ctx.Image()); err != nil {
		return nil
	}
	return buf.Bytes()
}
