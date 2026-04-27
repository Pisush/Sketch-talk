package renderer

import (
	"fmt"
	"image"

	"github.com/pisush/sketch-talk/internal/ai"
)

const (
	CanvasW = 1920
	CanvasH = 1080
)

// ZoneRect is the pixel bounding box of a named layout zone.
type ZoneRect struct {
	X, Y, W, H float64
}

// ZoneMap maps zone names to their pixel rectangles.
type ZoneMap map[ai.ZoneName]ZoneRect

// DefaultZoneMap returns the standard sketchnote zone layout.
func DefaultZoneMap() ZoneMap {
	return ZoneMap{
		ai.ZoneHeader:     {X: 0, Y: 0, W: CanvasW, H: 120},
		ai.ZoneMainLeft:   {X: 20, Y: 140, W: 880, H: 740},
		ai.ZoneMainRight:  {X: 1020, Y: 140, W: 880, H: 740},
		ai.ZoneMainCenter: {X: 20, Y: 140, W: 1880, H: 740},
		ai.ZoneFooter:     {X: 0, Y: 920, W: CanvasW, H: 160},
	}
}

// Resolve converts a SketchElement's zone-relative position to absolute pixel coords.
// Returns the element's bounding box in pixels.
func (zm ZoneMap) Resolve(el ai.SketchElement) (image.Rectangle, error) {
	zone, ok := zm[el.Zone]
	if !ok {
		return image.Rectangle{}, fmt.Errorf("unknown zone %q", el.Zone)
	}
	x := zone.X + el.RelX*zone.W
	y := zone.Y + el.RelY*zone.H
	w := el.W * zone.W
	h := el.H * zone.H
	return image.Rect(int(x), int(y), int(x+w), int(y+h)), nil
}
