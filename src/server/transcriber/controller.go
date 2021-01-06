package transcriber

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"server/transcriber/recognizers"
	"server/transcriber/recognizers/gcp"
	"server/transcriber/utils"
	"strings"
)

var state = State{}

// Start ...
func Start(encoderPath string, outputPath string, segmentsPath string) {

	state.encoderPath = encoderPath
	state.outputPath = outputPath
	state.segmentsPath = segmentsPath

	state.recognizer = &gcp.Adapter{}
	state.processing = false
	state.pruning = false

	state.playlistInfo = PlaylistInfo{
		Init: SegmentInfo{
			Filename: "",
		},
		Segments: make(map[string]SegmentInfo),
	}

	/* Creating the sub-dir for outputting transcription results */
	err := os.MkdirAll(state.outputPath, 0777)
	if err != nil {
		fmt.Printf("[Start] Could not create transcription output sub-dir in temporary output directory: %v", err)
		return
	}

	utils.SetInterval(processNewSegments, 1000, true)
	utils.SetInterval(pruneOldTranscripts, 1000, true)
}

func processNewSegments() {
	if state.processing {
		fmt.Println("[processSegments] still processing...")
		return
	}

	state.processing = true

	files, err := ioutil.ReadDir(state.segmentsPath)
	if err != nil {
		fmt.Println("[processSegments] could not read segment list")
		state.processing = false
		return
	}

	for _, fileInfo := range files {
		filename := fileInfo.Name()

		// Handle the init segment if it hasn't been handled yet
		if state.playlistInfo.Init.Filename == "" && strings.Contains(filename, "init") {
			fmt.Println("[processSegments] storing init filename: ", filename)
			state.playlistInfo.Init.Filename = filename
			continue
		}

		_, segmentKnown := state.playlistInfo.Segments[filename]
		if segmentKnown {
			// If it's not in an errored state, continue iterating
			// This effectively allows us to "retry" a failed transcription for a specific segment
			if state.playlistInfo.Segments[filename].State != "errored" {
				continue
			}
		}

		// TODO use WebVTT - no json condition needed
		isAudio := strings.Contains(filename, ".m4s") && !strings.Contains(filename, ".json")
		if isAudio {
			fmt.Printf("[processSegments] processing audio file: %s \n", filename)

			segment := SegmentInfo{
				Filename: filename,
				State:    "processing",
			}

			state.playlistInfo.Segments[filename] = segment

			filepath := fmt.Sprintf("%s/%s", state.segmentsPath, filename)
			err := processAudio(filepath)
			if err != nil {
				segment.State = "processed"
			} else {
				segment.State = "errored"
			}

			fmt.Printf("[processSegments] processed audio file: %s \n", filename)
		} else {
			// fmt.Println("[processSegments] ignoring non-audio file: ", filename)
		}
	}

	state.processing = false
}

func pruneOldTranscripts() {
	if state.pruning {
		fmt.Println("[pruneOldTranscripts] still pruning...")
		return
	}

	state.pruning = true

	files, err := ioutil.ReadDir(state.segmentsPath)
	if err != nil {
		fmt.Println("[pruneOldTranscripts] could not read segment list")
		state.pruning = false
		return
	}

	/* Scanning through the known segments - if a known segment doesn't exist in the latest files list, prune it */
	toDelete := make([]string, 0)

	for filename := range state.playlistInfo.Segments {
		found := false

		for _, fileInfo := range files {
			if filename == fileInfo.Name() {
				found = true
				break
			}
		}

		if found {
			// Not pruning this file
			continue
		}

		transcriptPath := fmt.Sprintf("%s/%s", state.outputPath, fmt.Sprintf("%s.json", filename))
		fmt.Println("[pruneOldTranscripts] removing: ", transcriptPath)

		/* Removing the file */
		os.Remove(transcriptPath)

		/* Queuing to remove the reference */
		toDelete = append(toDelete, filename)
	}

	for _, filename := range toDelete {
		delete(state.playlistInfo.Segments, filename)
	}

	state.pruning = false
}

func processAudio(segmentPath string) error {
	fmt.Println("[processAudio] for: ", segmentPath)
	_, segmentFilename := filepath.Split(segmentPath)

	// Reading the segment into memory
	mdat, err := ioutil.ReadFile(segmentPath)
	if err != nil {
		fmt.Println("[processAudio] failed to read file: ", err)
		return err
	}

	// Reading the init segment into memory
	initSegmentPath := fmt.Sprintf("%s/%s", state.segmentsPath, state.playlistInfo.Init.Filename)
	init, err := ioutil.ReadFile(initSegmentPath)
	if err != nil {
		fmt.Println("[processAudio] failed to read init segment: ", err)
		return err
	}

	blob := append(init, mdat...)

	/* Extracting the audio stream from mp4 and converting to ogg */
	cmd := exec.Command(
		state.encoderPath,
		"-i", "pipe:0",
		"-f", "opus",  // Providing a format hint since ffmpeg cannot detect the format through conventional means (e.g. filename extension sniffing)
		"-vn",
		"-acodec", "libopus",
		"-b:a", "64k",
		"-ar", "16000",
		"-ac", "1",
		"pipe:1",
	)

	var outb bytes.Buffer

	cmd.Stdin = bytes.NewReader(blob)
	cmd.Stdout = &outb

	err = cmd.Run()
	if err != nil {
		fmt.Printf("[processAudio] Could not extract audio stream, err: %v \n", err)
		return err
	}

	resp, err := state.recognizer.Input(outb.Bytes())
	if err != nil {
		fmt.Println("[processAudio] Error transcribing audio data: ", err)
	} else {
		fmt.Println("[processAudio] Successfully transcribed audio for segment: ", segmentPath)
		writeTranscriptionForSegment(resp, fmt.Sprintf("%s/%s", state.outputPath, segmentFilename))
	}

	return nil
}

func writeTranscriptionForSegment(data recognizers.Response, path string) error {
	fmt.Println("[writeTranscriptionForSegment] Transcription received: ", data)

	filepath := fmt.Sprintf("%v.json", path)
	raw, err := json.Marshal(data)
	if err != nil {
		fmt.Println("[writeTranscriptionForSegment] Could not convert speech response to byte array: ", err)
		return err
	}

	fmt.Printf("[writeTranscriptionForSegment] Writing transcription to %s", filepath)
	err = ioutil.WriteFile(filepath, raw, 0644)
	if err != nil {
		fmt.Printf("[writeTranscriptionForSegment] Could not write to path %v, error was %v \n", filepath, err)
		return err
	}

	return nil
}
