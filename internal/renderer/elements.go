package renderer

import (
	"image"
	"math"
	"math/rand"
	"strings"

	"github.com/fogleman/gg"
	"github.com/pisush/sketch-talk/internal/ai"
)

// drawBackground fills the canvas with the paper color and ruled horizontal lines.
func drawBackground(ctx *gg.Context) {
	bg := ColorBackground
	ctx.SetRGBA(bg.R, bg.G, bg.B, bg.A)
	ctx.Clear()

	ruled := ColorRuled
	ctx.SetRGBA(ruled.R, ruled.G, ruled.B, ruled.A)
	ctx.SetLineWidth(1)
	for y := 40.0; y < CanvasH; y += 40 {
		ctx.DrawLine(0, y, CanvasW, y)
		ctx.Stroke()
	}
}

// wobbleLine draws a hand-drawn wobbly line between two points.
func wobbleLine(ctx *gg.Context, x1, y1, x2, y2, jitter float64) {
	const segments = 8
	dx := (x2 - x1) / segments
	dy := (y2 - y1) / segments
	// perpendicular direction (normalized)
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return
	}
	px := -dy / length
	py := dx / length

	ctx.MoveTo(x1, y1)
	for i := 1; i <= segments; i++ {
		tx := x1 + float64(i)*dx
		ty := y1 + float64(i)*dy
		var offset float64
		if i < segments {
			offset = (rand.Float64()*2 - 1) * jitter
		}
		ctx.LineTo(tx+px*offset, ty+py*offset)
	}
	ctx.Stroke()
}

// wobbleRect draws a hand-drawn rounded rectangle outline up to `progress` of its perimeter.
func wobbleRect(ctx *gg.Context, x, y, w, h, jitter, progress float64) {
	perimeter := 2 * (w + h)
	dist := progress * perimeter

	// Draw sides in order: top, right, bottom (reversed), left (reversed)
	type segment struct{ x1, y1, x2, y2 float64 }
	sides := []segment{
		{x, y, x + w, y},
		{x + w, y, x + w, y + h},
		{x + w, y + h, x, y + h},
		{x, y + h, x, y},
	}
	sideLengths := []float64{w, h, w, h}

	consumed := 0.0
	for i, side := range sides {
		sLen := sideLengths[i]
		if consumed >= dist {
			break
		}
		portion := 1.0
		remaining := dist - consumed
		if remaining < sLen {
			portion = remaining / sLen
		}
		ex := side.x1 + (side.x2-side.x1)*portion
		ey := side.y1 + (side.y2-side.y1)*portion
		wobbleLine(ctx, side.x1, side.y1, ex, ey, jitter)
		consumed += sLen
	}
}

// drawBanner draws a full-width text banner (header/footer strips).
func drawBanner(ctx *gg.Context, el ai.SketchElement, bbox image.Rectangle, primary Color, progress float64) {
	c := ElementColor(el, primary)
	bw := float64(bbox.Dx())
	bh := float64(bbox.Dy())
	x := float64(bbox.Min.X)
	y := float64(bbox.Min.Y)

	// Fill wipes left-to-right
	fillW := bw * progress
	ctx.SetRGBA(c.R, c.G, c.B, 0.25)
	ctx.DrawRectangle(x, y, fillW, bh)
	ctx.Fill()

	// Text appears at progress > 0.5
	if progress > 0.5 {
		textProgress := (progress - 0.5) * 2
		setFont(ctx, true, FontSizeFor(el.Kind, el.Emphasis))
		ctx.SetRGBA(ColorText.R, ColorText.G, ColorText.B, textProgress)
		words := strings.Fields(el.Text)
		visible := int(math.Round(float64(len(words)) * textProgress))
		if visible > len(words) {
			visible = len(words)
		}
		ctx.DrawStringWrapped(strings.Join(words[:visible], " "),
			x+bw/2, y+bh/2, 0.5, 0.5, bw-20, 1.2, gg.AlignCenter)
	}
}

// drawBox draws a hand-drawn rounded box with text.
func drawBox(ctx *gg.Context, el ai.SketchElement, bbox image.Rectangle, primary Color, progress float64) {
	c := ElementColor(el, primary)
	x := float64(bbox.Min.X)
	y := float64(bbox.Min.Y)
	w := float64(bbox.Dx())
	h := float64(bbox.Dy())

	ctx.SetRGBA(c.R, c.G, c.B, 1.0)
	ctx.SetLineWidth(EmphasisStrokeWidth(el.Emphasis))
	wobbleRect(ctx, x+2, y+2, w-4, h-4, 2.5, progress)

	// Shadow outline at full
	if progress >= 1.0 {
		ctx.SetRGBA(c.R, c.G, c.B, 0.15)
		ctx.SetLineWidth(EmphasisStrokeWidth(el.Emphasis) + 1.5)
		ctx.DrawRoundedRectangle(x+4, y+4, w-4, h-4, 12)
		ctx.Stroke()

		ctx.SetRGBA(c.R, c.G, c.B, 0.10)
		ctx.DrawRoundedRectangle(x, y, w, h, 12)
		ctx.Fill()

		setFont(ctx, el.Emphasis >= 2, FontSizeFor(el.Kind, el.Emphasis))
		ctx.SetRGBA(ColorText.R, ColorText.G, ColorText.B, 1.0)
		ctx.DrawStringWrapped(el.Text, x+w/2, y+h/2, 0.5, 0.5, w-16, 1.2, gg.AlignCenter)
	}
}

// drawBubble draws a speech/thought bubble with text.
func drawBubble(ctx *gg.Context, el ai.SketchElement, bbox image.Rectangle, primary Color, progress float64) {
	c := ElementColor(el, primary)
	cx := float64(bbox.Min.X) + float64(bbox.Dx())/2
	cy := float64(bbox.Min.Y) + float64(bbox.Dy())/2
	rx := float64(bbox.Dx()) / 2
	ry := float64(bbox.Dy()) / 2

	// Trace arc from 0 to progress*2π
	endAngle := progress * 2 * math.Pi
	ctx.SetRGBA(c.R, c.G, c.B, 1.0)
	ctx.SetLineWidth(EmphasisStrokeWidth(el.Emphasis))
	ctx.DrawEllipticalArc(cx, cy, rx-4, ry-4, 0, endAngle)
	ctx.Stroke()

	// Tail spike appears at progress > 0.8
	if progress > 0.8 {
		p := (progress - 0.8) / 0.2
		tx := cx + rx*0.3
		ty := cy + ry
		ctx.MoveTo(tx, ty)
		ctx.LineTo(tx+20*p, ty+20*p)
		ctx.LineTo(tx-10*p, ty)
		ctx.Stroke()
	}

	if progress >= 1.0 {
		ctx.SetRGBA(c.R, c.G, c.B, 0.12)
		ctx.DrawEllipse(cx, cy, rx-4, ry-4)
		ctx.Fill()

		setFont(ctx, el.Emphasis >= 2, FontSizeFor(el.Kind, el.Emphasis))
		ctx.SetRGBA(ColorText.R, ColorText.G, ColorText.B, 1.0)
		ctx.DrawStringWrapped(el.Text, cx, cy, 0.5, 0.5, (rx-8)*2, 1.2, gg.AlignCenter)
	}
}

// drawText draws a text element (title, heading, bullet, quote) with word-by-word reveal.
func drawText(ctx *gg.Context, el ai.SketchElement, bbox image.Rectangle, primary Color, progress float64) {
	c := ElementColor(el, primary)
	x := float64(bbox.Min.X)
	y := float64(bbox.Min.Y)
	w := float64(bbox.Dx())

	bold := el.Kind == ai.KindTitle || el.Kind == ai.KindHeading || el.Emphasis >= 2
	setFont(ctx, bold, FontSizeFor(el.Kind, el.Emphasis))

	words := strings.Fields(el.Text)
	textProgress := progress
	if el.Kind == ai.KindTitle || el.Kind == ai.KindBanner {
		textProgress = progress
	}
	visible := int(math.Ceil(float64(len(words)) * textProgress))
	if visible > len(words) {
		visible = len(words)
	}

	ctx.SetRGBA(c.R, c.G, c.B, 1.0)
	ctx.DrawStringWrapped(strings.Join(words[:visible], " "),
		x, y, 0, 0, w, 1.3, gg.AlignLeft)

	// Bullet dot
	if el.Kind == ai.KindBullet && progress > 0.1 {
		ctx.SetRGBA(c.R, c.G, c.B, 1.0)
		ctx.DrawCircle(x-14, y+FontSizeFor(el.Kind, el.Emphasis)*0.5, 5)
		ctx.Fill()
	}

	// Wavy underline for titles/headings after text is done
	if (el.Kind == ai.KindTitle || el.Kind == ai.KindHeading) && progress >= 1.0 {
		uw := w * 0.6
		ux := x
		uy := y + FontSizeFor(el.Kind, el.Emphasis) + 6
		drawWavyLine(ctx, ux, uy, ux+uw, c)
	}
}

// drawWavyLine draws a hand-drawn wavy underline.
func drawWavyLine(ctx *gg.Context, x1, y, x2 float64, c Color) {
	ctx.SetRGBA(c.R, c.G, c.B, 0.8)
	ctx.SetLineWidth(2.5)
	amplitude := 4.0
	wavelength := 20.0
	steps := int((x2-x1)/wavelength*4) + 1
	ctx.MoveTo(x1, y)
	for i := 1; i <= steps; i++ {
		px := x1 + float64(i)*(x2-x1)/float64(steps)
		py := y + math.Sin(float64(i)*math.Pi*2/4)*amplitude
		ctx.LineTo(px, py)
	}
	ctx.Stroke()
}

// drawArrow draws a bezier arrow between two bounding boxes.
func drawArrow(ctx *gg.Context, from, to image.Rectangle, c Color, progress float64) {
	// Start from center of source, end at center of target.
	x1 := float64(from.Min.X+from.Max.X) / 2
	y1 := float64(from.Min.Y+from.Max.Y) / 2
	x2 := float64(to.Min.X+to.Max.X) / 2
	y2 := float64(to.Min.Y+to.Max.Y) / 2

	// Interpolate endpoint by progress.
	ex := x1 + (x2-x1)*progress
	ey := y1 + (y2-y1)*progress

	// Control points offset perpendicularly.
	midX := (x1 + ex) / 2
	midY := (y1 + ey) / 2
	dx := ex - x1
	dy := ey - y1
	length := math.Sqrt(dx*dx + dy*dy)
	if length > 0 {
		midX += -dy / length * 40
		midY += dx / length * 40
	}

	ctx.SetRGBA(c.R, c.G, c.B, 1.0)
	ctx.SetLineWidth(2.0)
	ctx.MoveTo(x1, y1)
	ctx.QuadraticTo(midX, midY, ex, ey)
	ctx.Stroke()

	// Arrowhead at progress > 0.9
	if progress >= 0.9 {
		arrowSize := 12.0
		angle := math.Atan2(ey-midY, ex-midX)
		ctx.MoveTo(ex, ey)
		ctx.LineTo(ex-arrowSize*math.Cos(angle-0.4), ey-arrowSize*math.Sin(angle-0.4))
		ctx.LineTo(ex-arrowSize*math.Cos(angle+0.4), ey-arrowSize*math.Sin(angle+0.4))
		ctx.ClosePath()
		ctx.Fill()
	}
}

// drawDivider draws a horizontal wavy divider line.
func drawDivider(ctx *gg.Context, el ai.SketchElement, bbox image.Rectangle, primary Color, progress float64) {
	c := ElementColor(el, primary)
	x := float64(bbox.Min.X)
	y := float64(bbox.Min.Y) + float64(bbox.Dy())/2
	w := float64(bbox.Dx()) * progress

	ctx.SetRGBA(c.R, c.G, c.B, 0.6)
	ctx.SetLineWidth(2)
	drawWavyLine(ctx, x, y, x+w, c)
}

// drawHighlight draws a highlight box behind existing text.
func drawHighlight(ctx *gg.Context, el ai.SketchElement, bbox image.Rectangle, _ Color, progress float64) {
	x := float64(bbox.Min.X)
	y := float64(bbox.Min.Y)
	w := float64(bbox.Dx()) * progress
	h := float64(bbox.Dy())

	ctx.SetRGBA(ColorHighlight.R, ColorHighlight.G, ColorHighlight.B, 0.7)
	ctx.DrawRoundedRectangle(x, y, w, h, 4)
	ctx.Fill()

	if progress >= 1.0 {
		setFont(ctx, el.Emphasis >= 2, FontSizeFor(el.Kind, el.Emphasis))
		ctx.SetRGBA(ColorText.R, ColorText.G, ColorText.B, 1.0)
		ctx.DrawStringWrapped(el.Text, x+float64(bbox.Dx())/2, y+h/2,
			0.5, 0.5, float64(bbox.Dx())-8, 1.2, gg.AlignCenter)
	}
}

// drawIcon draws a named icon as simple geometric shapes.
func drawIcon(ctx *gg.Context, el ai.SketchElement, bbox image.Rectangle, primary Color, progress float64) {
	c := ElementColor(el, primary)
	cx := float64(bbox.Min.X) + float64(bbox.Dx())/2
	cy := float64(bbox.Min.Y) + float64(bbox.Dy())/2
	r := math.Min(float64(bbox.Dx()), float64(bbox.Dy())) / 2 * 0.8

	ctx.SetRGBA(c.R, c.G, c.B, 1.0)
	ctx.SetLineWidth(2.5)

	// All icons are drawn as simple geometric constructions.
	endAngle := progress * 2 * math.Pi
	switch el.Icon {
	case "star":
		drawStar(ctx, cx, cy, r, progress)
	case "lightbulb":
		ctx.DrawEllipticalArc(cx, cy-r*0.1, r*0.7, r*0.7, 0, endAngle)
		ctx.Stroke()
		if progress >= 1.0 {
			ctx.DrawLine(cx-r*0.3, cy+r*0.6, cx+r*0.3, cy+r*0.6)
			ctx.Stroke()
		}
	case "checkmark":
		p := progress
		ctx.MoveTo(cx-r*0.5, cy)
		ctx.LineTo(cx-r*0.15, cy+r*0.4*p)
		if progress > 0.5 {
			ctx.LineTo(cx+r*0.5, cy-r*0.4*(progress-0.5)*2)
		}
		ctx.Stroke()
	case "person":
		ctx.DrawArc(cx, cy-r*0.4, r*0.3, 0, endAngle)
		ctx.Stroke()
		if progress >= 1.0 {
			ctx.DrawLine(cx, cy-r*0.1, cx, cy+r*0.5)
			ctx.Stroke()
			ctx.DrawLine(cx-r*0.4, cy+r*0.1, cx+r*0.4, cy+r*0.1)
			ctx.Stroke()
		}
	default:
		// Generic circle for unknown icons.
		ctx.DrawArc(cx, cy, r, 0, endAngle)
		ctx.Stroke()
	}
}

func drawStar(ctx *gg.Context, cx, cy, r, progress float64) {
	points := 5
	total := float64(points * 2)
	drawn := int(math.Round(progress * total))
	if drawn == 0 {
		return
	}
	var pts [][2]float64
	for i := 0; i < points*2; i++ {
		angle := float64(i)*math.Pi/float64(points) - math.Pi/2
		rad := r
		if i%2 == 1 {
			rad = r * 0.4
		}
		pts = append(pts, [2]float64{cx + rad*math.Cos(angle), cy + rad*math.Sin(angle)})
	}
	if drawn > len(pts) {
		drawn = len(pts)
	}
	ctx.MoveTo(pts[0][0], pts[0][1])
	for i := 1; i < drawn; i++ {
		ctx.LineTo(pts[i][0], pts[i][1])
	}
	if drawn == len(pts) {
		ctx.ClosePath()
	}
	ctx.Stroke()
}

// drawSparkle draws small decorative stars.
func drawSparkle(ctx *gg.Context, el ai.SketchElement, bbox image.Rectangle, primary Color, progress float64) {
	c := ElementColor(el, primary)
	ctx.SetRGBA(c.R, c.G, c.B, 0.6)
	ctx.SetLineWidth(1.5)
	cx := float64(bbox.Min.X) + float64(bbox.Dx())/2
	cy := float64(bbox.Min.Y) + float64(bbox.Dy())/2
	r := 8.0
	for i := 0; i < 4; i++ {
		if float64(i)/4 > progress {
			break
		}
		ox := (float64(i%2)*2 - 1) * 20
		oy := (float64(i/2)*2 - 1) * 15
		drawStar(ctx, cx+ox, cy+oy, r, 1.0)
	}
}
