package transcriber

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/fsnotify/fsnotify"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

var transcriptionHistory map[string]bool

// State ...
type State struct {
	encoder string
}

var state = State{}

// Transcriber ...
func Transcriber(encoder string) {
	state.encoder = encoder

	transcriptionHistory = make(map[string]bool)

	// creates a new file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("[transcriber] could not create new watcher: ", err)
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("[transcriber] error Getwd(), err: ", err)
		return
	}

	defer watcher.Close()

	done := make(chan bool)

	go func() {
		var ready = false

		for {
			select {
			// watch for events
			case event := <-watcher.Events:

				var shouldProcessAudio = ready && event.Op == fsnotify.Write && strings.Contains(event.Name, ".m4s") && !strings.Contains(event.Name, ".json")
				if shouldProcessAudio {
					segment := event.Name
					if !transcriptionHistory[segment] {
						transcriptionHistory[segment] = true
						processAudio(segment)
					} else {
						fmt.Println("[transcriber] Skipping processing of segment: ", segment)
					}
				}

				if event.Op == fsnotify.Create {

					// Hack, plz remove me
					if !ready {
						if strings.Contains(event.Name, "\\tmp\\content\\0") {

							fmt.Println("[transcriber] adding new watch folder: ", wd+"\\tmp\\content\\0\\")
							if err := watcher.Add(wd + "\\tmp\\content\\0\\"); err != nil {
								fmt.Println("[transcriber] error adding content watcher: ", err)
							}

							// TODO remove original watcher

							ready = true
						}
					}

				}

				// Cleanup out-of-window working files as
				// it ensures the ability to re-encode from working files if
				// a problem is encountered in the first pass
				if event.Op == fsnotify.Remove {
					cleanupForFragment(event.Name)
				}

			case err := <-watcher.Errors:
				fmt.Println("[transcriber] fsnotify error: ", err)
			}
		}
	}()

	if err := watcher.Add(wd + "\\tmp\\content\\"); err != nil {
		fmt.Println("[transcriber] error adding initial watcher", err)
	}

	<-done
}

func processAudio(segmentPath string) {
	fmt.Println("[processAudio] for: ", segmentPath)
	_, segmentFilename := filepath.Split(segmentPath)

	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("[processAudio] error Getwd(), err: ", err)
		return
	}

	// TODO parameterize
	err = fileExists("tmp")
	if err != nil {
		// TODO why 0777?
		os.Mkdir("tmp", 0777)
		fmt.Println("[processAudio] made tmp directory")
	}

	// Reading the segment into memory
	mdat, err := ioutil.ReadFile(segmentPath)
	if err != nil {
		fmt.Println("[processAudio] failed to read file: ", err)
		return
	}

	// Reading the init segment into memory
	// TODO refactor so we don't assume the lowest quality level is 0, and the filename pattern
	initSegmentPath := fmt.Sprintf("%v/tmp/content/0/init_0.mp4", wd)
	init, err := ioutil.ReadFile(initSegmentPath)
	if err != nil {
		fmt.Println("[processAudio] failed to read init segment: ", err)
		return
	}

	blob := append(init, mdat...)

	// TODO skip writing file, pass bytes directly to ffmpeg
	inputMP4Path := fmt.Sprintf("%v/tmp/convert/input_%v.mp4", wd, segmentFilename)

	err = ioutil.WriteFile(inputMP4Path, blob, 0777)
	if err != nil {
		fmt.Printf("[processAudio] Could not write input_%v.mp4, err: %v \n", segmentFilename, err)
		return
	}

	/* Extracting the audio stream from mp4 and converting to ogg */
	outputOGGPath := fmt.Sprintf("%v/tmp/convert/output_%v.ogg", wd, segmentFilename)

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
		return
	}

	data, err := ioutil.ReadFile(outputOGGPath)
	if err != nil {
		fmt.Println("[processAudio] Failed to read ogg file: ", outputOGGPath, err)
		return
	}

	ctx := context.Background()

	client, err := speech.NewClient(ctx)
	if err != nil {
		fmt.Println("[processAudio] Could not create speech client: ", err)
		return
	}

	// Detects speech in the audio file.
	resp, err := client.Recognize(ctx, &speechpb.RecognizeRequest{
		// TODO parameterize
		Config: &speechpb.RecognitionConfig{
			Encoding:                   speechpb.RecognitionConfig_OGG_OPUS,
			SampleRateHertz:            16000,
			LanguageCode:               "en-US",
			Model:                      "video",
			EnableWordTimeOffsets:      true,
			EnableAutomaticPunctuation: true, // This flag only works on English content
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: data},
		},
	})

	if err != nil {
		fmt.Println("[processAudio] Error transcribing audio data: ", err)
	} else {
		fmt.Println("[processAudio] Successfully transcribed audio for segment: ", segmentPath)
		writeTranscriptionForSegment(resp, segmentPath)
	}
}

func writeTranscriptionForSegment(data *speechpb.RecognizeResponse, path string) error {
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

func fileExists(path string) error {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		return err
	}

	return nil
}

func cleanupForFragment(fragFilepath string) {
	segmentDir, segmentFilename := filepath.Split(fragFilepath)

	transcriptionPath := fmt.Sprintf("%v/%v.json", segmentDir, segmentFilename)

	tmpMP4Path := fmt.Sprintf("tmp/convert/input_%v.mp4", segmentFilename)
	tmpOGGPath := fmt.Sprintf("tmp/convert/output_%v.ogg", segmentFilename)

	os.Remove(transcriptionPath)
	os.Remove(tmpMP4Path)
	os.Remove(tmpOGGPath)
}
