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
	encoder      string
	recognizer   recognizers.Adapter
	processing   bool
	pruning      bool
	playlistInfo PlaylistInfo
	dir          string
}
