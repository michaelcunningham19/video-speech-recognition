package gcp

import (
	"context"
	"errors"
	"fmt"
	"server/transcriber/recognizers"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

// Adapter ...
type Adapter struct {
}

// Init ...
func (a *Adapter) Init() {

}

// Input ...
func (a *Adapter) Input(audio []byte) (recognizers.Response, error) {
	ctx := context.Background()

	client, err := speech.NewClient(ctx)
	if err != nil {
		fmt.Println("[Input] Could not create speech client: ", err)
		return recognizers.Response{}, errors.New("100/Error preparing")
	}

	// Detects speech in the audio file.
	resp, err := client.Recognize(ctx, &speechpb.RecognizeRequest{
		// TODO parameterize
		Config: &speechpb.RecognitionConfig{
			Encoding:                   speechpb.RecognitionConfig_OGG_OPUS,
			SampleRateHertz:            16000,
			LanguageCode:               "en-US",
			Model:                      "video",
			UseEnhanced:                true,
			EnableWordTimeOffsets:      true,
			EnableAutomaticPunctuation: true, // This flag only works on English content
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: audio},
		},
	})

	return FromRecognizeResponse(resp), nil
}
