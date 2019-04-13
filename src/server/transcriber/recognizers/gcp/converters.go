package gcp

import (
	"server/transcriber/recognizers"

	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

// ToTimedWords ...
func ToTimedWords(words []*speechpb.WordInfo) []recognizers.TimedWord {
	// fmt.Println("[ToTimedWords] converting: ", words)

	result := make([]recognizers.TimedWord, 0)

	for _, word := range words {
		result = append(result, recognizers.TimedWord{
			Start: recognizers.PreciseTime{
				Seconds: word.StartTime.GetSeconds(),
				Nanos:   word.StartTime.GetNanos(),
			},
			End: recognizers.PreciseTime{
				Seconds: word.EndTime.GetSeconds(),
				Nanos:   word.EndTime.GetNanos(),
			},
			Word: word.Word,
		})
	}

	return result
}

// FromRecognizeResponse ...
func FromRecognizeResponse(resp *speechpb.RecognizeResponse) recognizers.Response {

	confidence := resp.Results[0].Alternatives[0].Confidence
	words := ToTimedWords(resp.Results[0].Alternatives[0].GetWords())

	return recognizers.Response{
		Confidence: confidence,
		Words:      words,
	}

}
