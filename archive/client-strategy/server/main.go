package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/gorilla/websocket"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

var ffmpeg = os.Getenv("FFMPEG_PATH")

const port = 13000

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// No `Origin` request checks needed
	CheckOrigin: func(h *http.Request) bool {
		return true
	},
}

var addr = flag.String("addr", fmt.Sprintf("localhost:%v", port), "http service address")

func main() {
	flag.Parse()
	http.HandleFunc("/speech-recognition", handler)

	fmt.Println("[main] listening on port: ", port)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	c, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("[handler] failed to set websocket upgrade: ", err)
		return
	}

	defer c.Close()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			fmt.Println("[handler] error when reading message: ", err)
			break
		}

		processAudio(message, c)
	}
}

func processAudio(raw []byte, conn *websocket.Conn) {
	var err error

	// TODO possible collisions?
	id := rand.Intn(100)

	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("[processAudio] error Getwd(), err: ", err)
		return
	}

	// TODO parameterize
	err = fileExists("tmp")
	if err != nil {
		// TODO why 0777?
		_ = os.Mkdir("tmp", 0777)
		fmt.Println("[processAudio] made tmp directory")
	}

	// TODO skip writing file, pass bytes directly to ffmpeg
	inputMP4Path := fmt.Sprintf("%v/tmp/input_%v.mp4", wd, id)

	err = ioutil.WriteFile(inputMP4Path, raw, 0777)
	if err != nil {
		fmt.Println("[processAudio] could not write bytes to disk, err: ", err)
		return
	}

	inputAACPath := fmt.Sprintf("%v/tmp/input_%v.aac", wd, id)

	// Extracting the audio stream from mp4 -> aac
	cmd := exec.Command(
		ffmpeg,
		"-i", inputMP4Path,
		"-vn",
		"-acodec", "copy",
		inputAACPath,
	)

	err = cmd.Run()
	if err != nil {
		fmt.Println("[processAudio] could not extract audio stream from input MP4: ", err)
		return
	}

	outputOGGPath := fmt.Sprintf("%v/tmp/output_%v.ogg", wd, id)

	// lossy audio is required to be encoded with libopus, according to API docs
	cmd = exec.Command(
		ffmpeg,
		"-i", inputAACPath,
		"-acodec", "libopus",
		"-b:a", "64000", // 64k
		"-ar", "16000", // 16kHz (required)
		"-ac", "1",
		outputOGGPath,
	)

	err = cmd.Run()
	if err != nil {
		fmt.Println("[processAudio] could not transcode audio stream from aac -> ogg opus: ", err)
		return
	}

	data, err := ioutil.ReadFile(outputOGGPath)
	if err != nil {
		fmt.Println("[processAudio] failed to read output OGG file: ", err)
		return
	}

	ctx := context.Background()

	client, err := speech.NewClient(ctx)
	if err != nil {
		fmt.Println("[processAudio] could not create speech client: ", err)
		return
	}

	// Detects speech in the audio file.
	resp, err := client.Recognize(ctx, &speechpb.RecognizeRequest{
		// TODO parameterize
		Config: &speechpb.RecognitionConfig{
			Encoding:                   speechpb.RecognitionConfig_OGG_OPUS,
			SampleRateHertz:            16000,
			LanguageCode:               "en-US",
			EnableWordTimeOffsets:      true,
			EnableAutomaticPunctuation: true, // This flag only works on English content
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: data},
		},
	})

	if err != nil {
		fmt.Println("[processAudio] error transcribing audio data: ", err)
		return
	}

	fmt.Println("[processAudio] successfully transcribed audio")
	cleanupForID(id)
	conn.WriteJSON(resp)
}

func fileExists(path string) error {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		return err
	}

	return nil
}

func cleanupForID(tmpID int) {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("[cleanupForID] error Getwd(), err: ", err)
		return
	}

	inputAACPath := fmt.Sprintf("%v/tmp/input_%v.aac", wd, tmpID)
	inputMP4Path := fmt.Sprintf("%v/tmp/input_%v.mp4", wd, tmpID)
	outputOGGPath := fmt.Sprintf("%v/tmp/output_%v.ogg", wd, tmpID)

	os.Remove(inputAACPath)
	os.Remove(inputMP4Path)
	os.Remove(outputOGGPath)
}
