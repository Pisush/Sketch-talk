package renderer

import "github.com/pisush/sketch-talk/internal/ai"

// RGBA color values (r, g, b, a) all 0.0–1.0.
type Color struct{ R, G, B, A float64 }

var (
	ColorBackground = Color{0.98, 0.98, 0.96, 1.0} // #FAFAF5
	ColorRuled      = Color{0.91, 0.89, 0.85, 0.3} // #E8E4D9
	ColorText       = Color{0.10, 0.10, 0.10, 1.0} // #1A1A1A
	ColorBorder     = Color{0.20, 0.20, 0.20, 1.0} // #333333
	ColorHighlight  = Color{1.0, 0.88, 0.40, 0.85} // #FFE066

	PaletteBlue   = Color{0.23, 0.51, 0.82, 0.75}
	PaletteGreen  = Color{0.18, 0.65, 0.38, 0.75}
	PaletteRed    = Color{0.82, 0.22, 0.22, 0.75}
	PalettePurple = Color{0.53, 0.22, 0.82, 0.75}
	PaletteOrange = Color{0.92, 0.52, 0.13, 0.75}
)

// PrimaryColorFor maps a color name (from Claude) to a palette Color.
func PrimaryColorFor(name string) Color {
	switch name {
	case "green":
		return PaletteGreen
	case "red":
		return PaletteRed
	case "purple":
		return PalettePurple
	case "orange":
		return PaletteOrange
	default:
		return PaletteBlue
	}
}

// EmphasisStrokeWidth returns the line width for a given emphasis level (1–3).
func EmphasisStrokeWidth(emphasis int) float64 {
	switch emphasis {
	case 3:
		return 4.0
	case 2:
		return 2.5
	default:
		return 1.8
	}
}

// ElementColor returns the draw color for an element, falling back to primary.
func ElementColor(el ai.SketchElement, primary Color) Color {
	switch el.Color {
	case "blue":
		return PaletteBlue
	case "green":
		return PaletteGreen
	case "red":
		return PaletteRed
	case "purple":
		return PalettePurple
	case "orange":
		return PaletteOrange
	case "yellow":
		return ColorHighlight
	default:
		return primary
	}
}

// FontSizeFor returns point size for an element kind.
func FontSizeFor(kind ai.ElementKind, emphasis int) float64 {
	switch kind {
	case ai.KindTitle, ai.KindBanner:
		if emphasis >= 3 {
			return 52
		}
		return 42
	case ai.KindHeading:
		return 32
	case ai.KindQuote:
		return 28
	case ai.KindBullet:
		return 22
	case ai.KindBox, ai.KindBubble, ai.KindHighlight:
		return 20
	default:
		return 18
	}
}
