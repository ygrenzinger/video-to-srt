package grok

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"video-to-srt/internal/subtitles"
	"video-to-srt/internal/transcription/transport"
)

const DefaultModel = "grok-transcribe-1"

type Duration = time.Duration

type Provider struct {
	URL         string
	Client      *http.Client
	Getenv      func(string) string
	RetryDelays []Duration
	Sleep       func(Duration)
}

func (p Provider) Transcribe(ctx context.Context, audioPath, outputPath, model string) error {
	return transport.Transcribe(ctx, transport.Request{
		ProviderName: "grok",
		URL:          p.URL,
		DefaultURL:   "https://api.x.ai/v1/stt",
		APIKeyEnv:    "XAI_API_KEY",
		Model:        model,
		DefaultModel: DefaultModel,
		AudioPath:    audioPath,
		OutputPath:   outputPath,
		FormFields: []transport.FormField{
			{Name: "response_format", Value: "verbose_json"},
			{Name: "timestamp_granularities[]", Value: "word"},
		},
		DecodeCues:  decodeCues,
		Client:      p.Client,
		Getenv:      p.Getenv,
		RetryDelays: p.RetryDelays,
		Sleep:       p.Sleep,
	})
}

func decodeCues(reader io.Reader) ([]subtitles.Cue, error) {
	var result map[string]any
	if err := json.NewDecoder(reader).Decode(&result); err != nil {
		return nil, fmt.Errorf("grok transcription response was not JSON: %w", err)
	}
	return resultToSubtitleCues(result)
}

func resultToSubtitleCues(result map[string]any) ([]subtitles.Cue, error) {
	if raw, ok := result["segments"].([]any); ok {
		cues := []subtitles.Cue{}
		for _, item := range raw {
			segment, ok := item.(map[string]any)
			if !ok {
				continue
			}
			text := strings.TrimSpace(fmt.Sprint(segment["text"]))
			if text == "" || text == "<nil>" {
				continue
			}
			start, ok1 := numberMS(segment["start"])
			end, ok2 := numberMS(segment["end"])
			if !ok1 || !ok2 {
				return nil, errors.New("grok returned a segment with invalid timestamp")
			}
			if end <= start {
				continue
			}
			cues = append(cues, subtitles.Cue{StartMS: start, EndMS: end, Text: text})
		}
		if len(cues) > 0 {
			return cues, nil
		}
	}
	if raw, ok := result["words"].([]any); ok {
		words := []map[string]any{}
		for _, item := range raw {
			if word, ok := item.(map[string]any); ok {
				words = append(words, word)
			}
		}
		cues := wordsToSubtitleCues(words)
		if len(cues) > 0 {
			return cues, nil
		}
	}
	return nil, errors.New("grok returned no timestamped transcription cues")
}

func wordsToSubtitleCues(words []map[string]any) []subtitles.Cue {
	cues := []subtitles.Cue{}
	current := []map[string]any{}
	flush := func() {
		if len(current) == 0 {
			return
		}
		cues = append(cues, cueFromWords(current))
		current = nil
	}
	for _, word := range words {
		text := strings.TrimSpace(firstString(word["word"], word["text"]))
		if text == "" {
			continue
		}
		word["text"] = text
		if len(current) > 0 {
			start, _ := numberSeconds(current[0]["start"])
			end, _ := numberSeconds(word["end"])
			tooLong := end-start > 7
			texts := []string{}
			for _, item := range append(current, word) {
				texts = append(texts, fmt.Sprint(item["text"]))
			}
			tooManyChars := len(strings.Join(texts, " ")) > 84
			sentenceDone := strings.HasSuffix(strings.TrimRight(fmt.Sprint(current[len(current)-1]["text"]), " "), ".") || strings.HasSuffix(fmt.Sprint(current[len(current)-1]["text"]), "?") || strings.HasSuffix(fmt.Sprint(current[len(current)-1]["text"]), "!")
			if tooLong || tooManyChars || sentenceDone {
				flush()
			}
		}
		current = append(current, word)
	}
	flush()
	return cues
}

func cueFromWords(words []map[string]any) subtitles.Cue {
	start, _ := numberMS(words[0]["start"])
	end, _ := numberMS(words[len(words)-1]["end"])
	texts := []string{}
	for _, word := range words {
		texts = append(texts, fmt.Sprint(word["text"]))
	}
	return subtitles.Cue{StartMS: start, EndMS: end, Text: strings.Join(texts, " ")}
}

func numberSeconds(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		f, err := strconv.ParseFloat(fmt.Sprint(v), 64)
		return f, err == nil
	}
}

func numberMS(v any) (int, bool) {
	f, ok := numberSeconds(v)
	if !ok {
		return 0, false
	}
	return int(f*1000 + math.Copysign(0.5, f)), true
}

func firstString(values ...any) string {
	for _, v := range values {
		if v != nil {
			s := fmt.Sprint(v)
			if s != "<nil>" {
				return s
			}
		}
	}
	return ""
}
