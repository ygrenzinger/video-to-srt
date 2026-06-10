package voxtral

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"video-to-srt/internal/subtitles"
	"video-to-srt/internal/transcription/transport"
)

const DefaultModel = "voxtral-mini-latest"

type Duration = time.Duration

type Provider struct {
	URL         string
	Client      *http.Client
	Getenv      func(string) string
	RetryDelays []Duration
	Sleep       func(Duration)
}

func (p Provider) Transcribe(ctx context.Context, audioPath, outputPath, model string) ([]subtitles.Cue, error) {
	return transport.Transcribe(ctx, transport.Request{
		ProviderName: "voxtral",
		URL:          p.URL,
		DefaultURL:   "https://api.mistral.ai/v1/audio/transcriptions",
		APIKeyEnv:    "MISTRAL_API_KEY",
		Model:        model,
		DefaultModel: DefaultModel,
		AudioPath:    audioPath,
		OutputPath:   outputPath,
		FormFields:   []transport.FormField{{Name: "timestamp_granularities", Value: "segment"}},
		DecodeCues:   decodeCues,
		Client:       p.Client,
		Getenv:       p.Getenv,
		RetryDelays:  p.RetryDelays,
		Sleep:        p.Sleep,
	})
}

type response struct {
	Segments []segment `json:"segments"`
}

type segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

func decodeCues(reader io.Reader) ([]subtitles.Cue, error) {
	var result response
	if err := json.NewDecoder(reader).Decode(&result); err != nil {
		return nil, fmt.Errorf("voxtral transcription response was not JSON: %w", err)
	}
	return result.subtitleCues()
}

func (r response) subtitleCues() ([]subtitles.Cue, error) {
	cues := []subtitles.Cue{}
	for _, segment := range r.Segments {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}
		start := int(segment.Start * 1000)
		end := int(segment.End * 1000)
		if end <= start {
			continue
		}
		cues = append(cues, subtitles.Cue{StartMS: start, EndMS: end, Text: text})
	}
	if len(cues) == 0 {
		return nil, errors.New("voxtral returned no usable timestamped segments")
	}
	return cues, nil
}
