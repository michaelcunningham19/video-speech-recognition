package recognizers

// PreciseTime ...
type PreciseTime struct {
	Seconds int64 `json:"seconds"`
	Nanos   int32 `json:"nanos"`
}

// TimedWord ...
type TimedWord struct {
	Start PreciseTime `json:"start"`
	End   PreciseTime `json:"end"`
	Word  string      `json:"word"`
}

// Response ...
type Response struct {
	Words      []TimedWord `json:"words"`
	Confidence float32     `json:"confidence"`
}

// Adapter ...
type Adapter interface {
	Init()
	Input(audio []byte) (Response, error)
}
