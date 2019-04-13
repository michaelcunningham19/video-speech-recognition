package main

// Will read the new segments for the lowest quality segment, converting to the required format and sending off to the transcriber at gcp
//
// phase one (complete): will do simple json output w/ client polling and processing
// phase two (todo)    : will do live webvtt, letting hls.js do everything on the client side
//

import (
	"os"
	"server/encoder"
	"server/transcriber"
)

var ffmpeg = os.Getenv("FFMPEG_PATH")

func main() {
	// TODO refactor name
	go transcriber.Transcriber(ffmpeg)

	encoder.Start(ffmpeg)
}
