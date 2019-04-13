package transcriber

import (
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
func Start(encoder string) {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("[processAudio] error Getwd(), err: ", err)
		panic(err)
	}

	state.encoder = encoder
	state.recognizer = &gcp.Adapter{}
	state.processing = false
	state.pruning = false
	state.dir = wd
	state.playlistInfo = PlaylistInfo{
		Init: SegmentInfo{
			Filename: "",
		},
		Segments: make(map[string]SegmentInfo),
	}

	// Preparing the convert tmp dir
	err = os.Mkdir(fmt.Sprintf("%s/encoder/tmp/convert", wd), 0644)
	if err != nil {
		fmt.Println("[Start] could not create tmp convert dir, retrying..", err)
		utils.SetTimeout(func() {
			Start(encoder)
		}, 1000)
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

	files, err := ioutil.ReadDir(fmt.Sprintf("%s/encoder/tmp/content/0", state.dir))
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
			fmt.Println("[processSegments] processing audio file... ", filename)

			segment := SegmentInfo{
				Filename: filename,
				State:    "processing",
			}

			state.playlistInfo.Segments[filename] = segment

			filepath := fmt.Sprintf("%s/encoder/tmp/content/0/%s", state.dir, filename)
			err := processAudio(filepath)
			if err != nil {
				segment.State = "processed"
			} else {
				segment.State = "errored"
			}

			fmt.Println("[processSegments] processed audio file: ", filename)
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

	files, err := ioutil.ReadDir(fmt.Sprintf("%s/encoder/tmp/content/0", state.dir))
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

		transcriptPath := fmt.Sprintf("%s/encoder/tmp/content/0/%s", state.dir, fmt.Sprintf("%s.json", filename))
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
	initSegmentPath := fmt.Sprintf("%s/encoder/tmp/content/0/%s", state.dir, state.playlistInfo.Init.Filename)
	init, err := ioutil.ReadFile(initSegmentPath)
	if err != nil {
		fmt.Println("[processAudio] failed to read init segment: ", err)
		return err
	}

	blob := append(init, mdat...)

	// TODO skip writing file, pass bytes directly to ffmpeg
	inputMP4Path := fmt.Sprintf("%s/encoder/tmp/convert/input_%v.mp4", state.dir, segmentFilename)

	err = ioutil.WriteFile(inputMP4Path, blob, 0777)
	if err != nil {
		fmt.Printf("[processAudio] Could not write input_%v.mp4, err: %v \n", segmentFilename, err)
		return err
	}

	/* Extracting the audio stream from mp4 and converting to ogg */
	outputOGGPath := fmt.Sprintf("%s/encoder/tmp/convert/output_%v.ogg", state.dir, segmentFilename)

	cmd := exec.Command(
		state.encoder,
		"-i", inputMP4Path,
		"-vn",
		"-acodec", "libopus",
		"-b:a", "64k",
		"-ar", "16000",
		"-ac", "1",
		outputOGGPath,
	)

	err = cmd.Run()
	if err != nil {
		fmt.Printf("[processAudio] Could not extract audio stream from path %v , err: %v \n", inputMP4Path, err)
		return err
	}

	data, err := ioutil.ReadFile(outputOGGPath)
	if err != nil {
		fmt.Println("[processAudio] Failed to read ogg file: ", outputOGGPath, err)
		return err
	}

	resp, err := state.recognizer.Input(data)
	if err != nil {
		fmt.Println("[processAudio] Error transcribing audio data: ", err)
	} else {
		fmt.Println("[processAudio] Successfully transcribed audio for segment: ", segmentPath)
		writeTranscriptionForSegment(resp, segmentPath)
	}

	cleanupForFragment(segmentPath)

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

	err = ioutil.WriteFile(filepath, raw, 0644)
	if err != nil {
		fmt.Printf("[writeTranscriptionForSegment] Could not write to path %v, error was %v \n", filepath, err)
		return err
	}

	return nil
}

func cleanupForFragment(fragFilepath string) {
	_, segmentFilename := filepath.Split(fragFilepath)

	tmpMP4Path := fmt.Sprintf("%s/encoder/tmp/convert/input_%v.mp4", state.dir, segmentFilename)
	tmpOGGPath := fmt.Sprintf("%s/encoder/tmp/convert/output_%v.ogg", state.dir, segmentFilename)

	os.Remove(tmpMP4Path)
	os.Remove(tmpOGGPath)
}
