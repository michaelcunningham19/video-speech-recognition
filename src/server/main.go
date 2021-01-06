package main

// Will read the new segments for the lowest quality segment, converting to the required format and sending off to the transcriber at gcp
//
// phase one (complete): will do simple json output w/ client polling and processing
// phase two (todo)    : will do live webvtt, letting the player do everything on the client side in a spec compliant way
//

import (
	"fmt"
	"os"
	"server/encoder"
	"server/transcriber"
)

var ffmpegPath = os.Getenv("FFMPEG_PATH")
const temporaryOutputDirName = "_tmp"

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("[main] Error Getwd(), err: ", err)
		panic(err)
	}

	var temporaryOutputDirPath = fmt.Sprintf("%s/%s", wd, temporaryOutputDirName)

	err = os.RemoveAll(fmt.Sprintf("%s/%s", "./", temporaryOutputDirName))
	if err != nil {
		fmt.Println("[main] Could not remove pre-existing temporary directory: ", err)
	}

	go transcriber.Start(
		ffmpegPath,
		fmt.Sprintf("%s/%s", temporaryOutputDirPath, "text"),  // Transcriber will output to /_tmp/text
		fmt.Sprintf("%s/%s", temporaryOutputDirPath, "0"),     // Transcriber will reference media segments that will exist in /_tmp/0
	)

	encoder.Start(
		ffmpegPath,
		temporaryOutputDirPath,  // Encoder will output to /_tmp
	)
}
