package renderer

import (
	"fmt"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/pisush/sketch-talk/assets"
	"golang.org/x/image/font"
)

var (
	fontRegular *truetype.Font
	fontBold    *truetype.Font
	fontCaption *truetype.Font
)

func loadFonts() error {
	var err error
	fontRegular, err = loadTTF("fonts/Caveat-Regular.ttf")
	if err != nil {
		return fmt.Errorf("load Caveat-Regular: %w", err)
	}
	fontBold, err = loadTTF("fonts/Caveat-Bold.ttf")
	if err != nil {
		return fmt.Errorf("load Caveat-Bold: %w", err)
	}
	fontCaption, err = loadTTF("fonts/PatrickHand-Regular.ttf")
	if err != nil {
		return fmt.Errorf("load PatrickHand: %w", err)
	}
	return nil
}

func loadTTF(path string) (*truetype.Font, error) {
	data, err := assets.FontFS.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return truetype.Parse(data)
}

// setFont sets the appropriate font face on a gg context.
func setFont(ctx *gg.Context, bold bool, size float64) {
	f := fontRegular
	if bold {
		f = fontBold
	}
	face := truetype.NewFace(f, &truetype.Options{
		Size: size,
		DPI:  96,
		Hinting: font.HintingFull,
	})
	ctx.SetFontFace(face)
}

// setCaptionFont sets the caption/alternate font.
func setCaptionFont(ctx *gg.Context, size float64) {
	face := truetype.NewFace(fontCaption, &truetype.Options{
		Size: size,
		DPI:  96,
		Hinting: font.HintingFull,
	})
	ctx.SetFontFace(face)
}
