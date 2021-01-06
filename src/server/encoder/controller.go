package encoder

import (
	"fmt"
	"os"
	"os/exec"
	"server/encoder/strategies"
)

// Start ...
func Start(encoderPath string, outputPath string) {

	streamSource := "https://live.corusdigitaldev.com/groupd/live/49a91e7f-1023-430f-8d66-561055f3d0f7/live.isml/live-audio_1=96000-video=2499968.m3u8" // Path to video input source
	streamOutputPath := outputPath + "/%v/playlist.m3u8"
	segmentFilenamePattern := outputPath + "/%v/%04d.m4s"  // Deliberate template variable for ffmpeg's use.

	mode := "x264"

	/* Default strategy - no-op */
	strategy := func(string, string, string, string) *exec.Cmd {
		return nil
	}

	if mode == "nvenc" {
		strategy = strategies.NVENC
	} else {
		strategy = strategies.X264
	}

	cmd := strategy(encoderPath, streamSource, segmentFilenamePattern, streamOutputPath)

	//
	// TODO better logging
	//
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("[main] Could not start encoder, err: ", err)
	}
}
