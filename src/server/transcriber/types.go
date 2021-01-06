package transcriber

import (
	"server/transcriber/recognizers"
)

// SegmentInfo ...
type SegmentInfo struct {
	Filename string
	State    string
}

// PlaylistInfo ...
type PlaylistInfo struct {
	Init     SegmentInfo
	Segments map[string]SegmentInfo
}

// State ...
type State struct {
	encoderPath  string
	outputPath   string
	segmentsPath string
	recognizer   recognizers.Adapter
	processing   bool
	pruning      bool
	playlistInfo PlaylistInfo
}
