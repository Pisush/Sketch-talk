package renderer

import (
	"image"
	"time"

	"github.com/pisush/sketch-talk/internal/ai"
)

const (
	animFrames = 20              // number of frames per element animation
	framePause = 50 * time.Millisecond // ~20fps
)

// AnimJob is a queued animation for a single new element.
type AnimJob struct {
	Element ai.SketchElement
	BBox    image.Rectangle
	Primary Color
	// FromBBox and ToBBox are set for arrow elements.
	FromBBox image.Rectangle
	ToBBox   image.Rectangle
}

// Animator drives stroke-by-stroke element animations and pushes PNG frames.
type Animator struct {
	canvas  *Canvas
	jobs    chan AnimJob
	frameCh chan<- []byte // outgoing PNG frames to WebSocket hub
	done    chan struct{}
}

// NewAnimator creates an Animator. frameCh receives PNG bytes after each frame.
func NewAnimator(canvas *Canvas, frameCh chan<- []byte) *Animator {
	a := &Animator{
		canvas:  canvas,
		jobs:    make(chan AnimJob, 64),
		frameCh: frameCh,
		done:    make(chan struct{}),
	}
	go a.run()
	return a
}

// Enqueue adds an element animation to the queue.
func (a *Animator) Enqueue(job AnimJob) {
	a.jobs <- job
}

// Stop signals the animator to finish current work and exit.
func (a *Animator) Stop() {
	close(a.jobs)
	<-a.done
}

// run is the animation loop — one goroutine, sequential jobs.
func (a *Animator) run() {
	defer close(a.done)
	for job := range a.jobs {
		a.animate(job)
	}
}

func (a *Animator) animate(job AnimJob) {
	el := job.Element
	bbox := job.BBox
	primary := job.Primary

	for frame := 0; frame <= animFrames; frame++ {
		progress := float64(frame) / float64(animFrames)

		a.canvas.mu.Lock()

		// Redraw everything already committed.
		drawBackground(a.canvas.ctx)
		for _, drawn := range a.canvas.committed {
			a.drawElement(drawn.Spec, drawn.BBox, drawn.Primary, drawn.FromBBox, drawn.ToBBox, 1.0)
		}

		// Draw the in-progress element at current progress.
		a.drawElement(el, bbox, primary, job.FromBBox, job.ToBBox, progress)

		png := a.canvas.encodePNG()
		a.canvas.mu.Unlock()

		select {
		case a.frameCh <- png:
		default:
		}

		if frame < animFrames {
			time.Sleep(framePause)
		}
	}

	// Commit the completed element.
	a.canvas.mu.Lock()
	a.canvas.committed = append(a.canvas.committed, DrawnElement{
		Spec:     el,
		BBox:     bbox,
		Primary:  primary,
		FromBBox: job.FromBBox,
		ToBBox:   job.ToBBox,
	})
	a.canvas.mu.Unlock()
}

// drawElement dispatches to the correct draw function based on element kind.
func (a *Animator) drawElement(el ai.SketchElement, bbox image.Rectangle, primary Color, fromBBox, toBBox image.Rectangle, progress float64) {
	ctx := a.canvas.ctx
	switch el.Kind {
	case ai.KindBanner:
		drawBanner(ctx, el, bbox, primary, progress)
	case ai.KindBox:
		drawBox(ctx, el, bbox, primary, progress)
	case ai.KindBubble:
		drawBubble(ctx, el, bbox, primary, progress)
	case ai.KindTitle, ai.KindHeading, ai.KindBullet, ai.KindQuote, ai.KindHighlight:
		if el.Kind == ai.KindHighlight {
			drawHighlight(ctx, el, bbox, primary, progress)
		} else {
			drawText(ctx, el, bbox, primary, progress)
		}
	case ai.KindArrow:
		drawArrow(ctx, fromBBox, toBBox, ElementColor(el, primary), progress)
	case ai.KindDivider:
		drawDivider(ctx, el, bbox, primary, progress)
	case ai.KindIcon:
		drawIcon(ctx, el, bbox, primary, progress)
	case ai.KindSparkle:
		drawSparkle(ctx, el, bbox, primary, progress)
	}
}
