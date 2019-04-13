package encoder

import (
	"fmt"
	"os"
	"os/exec"
	"server/encoder/strategies"
)

// Start ...
func Start(encoder string) {

	source := "" // Path to video input source
	outputDirectory := "./tmp/content/"
	outputPath := outputDirectory + "%v/playlist.m3u8"

	segmentFilename := outputDirectory + "%v/%04d.m4s"

	mode := "x264"

	strategy := func(string, string, string, string) *exec.Cmd {
		return nil
	}

	if mode == "nvenc" {
		strategy = strategies.NVENC
	} else {
		strategy = strategies.X264
	}

	cmd := strategy(encoder, source, segmentFilename, outputPath)

	//
	// TODO better logging
	//

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("[main] could not start encoder, err: ", err)
	}

}
