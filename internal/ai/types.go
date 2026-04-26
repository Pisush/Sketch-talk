package ai

// ElementKind enumerates all drawable sketchnote primitives.
type ElementKind string

const (
	KindTitle     ElementKind = "title"
	KindHeading   ElementKind = "heading"
	KindBullet    ElementKind = "bullet"
	KindBox       ElementKind = "box"
	KindBubble    ElementKind = "bubble"
	KindArrow     ElementKind = "arrow"
	KindIcon      ElementKind = "icon"
	KindDivider   ElementKind = "divider"
	KindQuote     ElementKind = "quote"
	KindHighlight ElementKind = "highlight"
	KindBanner    ElementKind = "banner"
	KindSparkle   ElementKind = "sparkle"
)

// ZoneName identifies a named layout region on the canvas.
type ZoneName string

const (
	ZoneHeader     ZoneName = "header"
	ZoneMainLeft   ZoneName = "main_left"
	ZoneMainRight  ZoneName = "main_right"
	ZoneMainCenter ZoneName = "main_center"
	ZoneFooter     ZoneName = "footer"
)

// SketchElement is the unit Claude emits for every visual element.
type SketchElement struct {
	ID       string      `json:"id"`
	Kind     ElementKind `json:"kind"`
	Zone     ZoneName    `json:"zone"`
	RelX     float64     `json:"rel_x"` // 0.0–1.0 within zone
	RelY     float64     `json:"rel_y"`
	W        float64     `json:"w"` // fraction of zone width
	H        float64     `json:"h"` // fraction of zone height
	Text     string      `json:"text,omitempty"`
	FromID   string      `json:"from_id,omitempty"`
	ToID     string      `json:"to_id,omitempty"`
	Icon     string      `json:"icon,omitempty"`
	Emphasis int         `json:"emphasis"` // 1–3
	Color    string      `json:"color,omitempty"`
}

// SketchSkeleton is the full initial layout Claude returns after slide analysis.
type SketchSkeleton struct {
	TalkTheme    string          `json:"talk_theme"`
	PrimaryColor string          `json:"primary_color"`
	Elements     []SketchElement `json:"elements"`
	KeyTerms     []string        `json:"key_terms"`
}

// TranscriptUpdate is what Claude returns for each live transcript chunk.
type TranscriptUpdate struct {
	AddElements    []SketchElement `json:"add_elements"`
	RemoveIDs      []string        `json:"remove_ids,omitempty"`
	UpdateElements []SketchElement `json:"update_elements,omitempty"`
}
