package config

// TalkConfig holds all metadata and runtime settings for a sketchnote session.
type TalkConfig struct {
	ConferenceName string
	ConferenceTag  string // e.g. "#gophercon2025"
	SpeakerName    string
	SpeakerHandle  string // e.g. "@gopher"
	TalkTitle      string
	SlidesPath     string

	AnthropicAPIKey string
	OpenAIAPIKey    string

	OutputPath       string
	ListenAddr       string
	AudioDeviceIndex int
	ChunkSeconds     int
	OverlapSeconds   int
	JPEGQuality      int // 0 = PNG, 1-100 = JPEG
}

func Default() *TalkConfig {
	return &TalkConfig{
		OutputPath:       "./sketchnote.png",
		ListenAddr:       ":8080",
		AudioDeviceIndex: -1,
		ChunkSeconds:     25,
		OverlapSeconds:   3,
		JPEGQuality:      0,
	}
}
