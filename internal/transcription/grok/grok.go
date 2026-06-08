package grok

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"video-to-srt/internal/subtitles"
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
	apiKey := p.getenv("XAI_API_KEY")
	if apiKey == "" {
		return errors.New("missing required environment variable: XAI_API_KEY")
	}
	if model == "" {
		model = DefaultModel
	}
	var lastErr error
	delays := p.retryDelays()
	for attempt := 0; attempt <= len(delays); attempt++ {
		err := p.transcribeOnce(ctx, audioPath, outputPath, model, apiKey)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt == len(delays) || !isRetryable(err) {
			return err
		}
		p.sleep(delays[attempt])
	}
	return lastErr
}

func (p Provider) transcribeOnce(ctx context.Context, audioPath, outputPath, model, apiKey string) error {
	bodyReader, bodyWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(bodyWriter)
	go func() {
		defer bodyWriter.Close()
		defer multipartWriter.Close()
		if err := multipartWriter.WriteField("model", model); err != nil {
			_ = bodyWriter.CloseWithError(err)
			return
		}
		if err := multipartWriter.WriteField("response_format", "verbose_json"); err != nil {
			_ = bodyWriter.CloseWithError(err)
			return
		}
		if err := multipartWriter.WriteField("timestamp_granularities[]", "word"); err != nil {
			_ = bodyWriter.CloseWithError(err)
			return
		}
		file, err := os.Open(audioPath)
		if err != nil {
			_ = bodyWriter.CloseWithError(err)
			return
		}
		defer file.Close()
		part, err := multipartWriter.CreateFormFile("file", filepath.Base(audioPath))
		if err != nil {
			_ = bodyWriter.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			_ = bodyWriter.CloseWithError(err)
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url(), bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	resp, err := p.client().Do(req)
	if err != nil {
		return providerError{message: "grok transcription failed: " + err.Error(), retryable: true, err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return providerError{message: fmt.Sprintf("grok transcription failed: HTTP %d", resp.StatusCode), statusCode: resp.StatusCode, retryable: resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500}
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("grok transcription response was not JSON: %w", err)
	}
	cues, err := resultToSubtitleCues(result)
	if err != nil {
		return err
	}
	return subtitles.AtomicWriteSRT(outputPath, cues)
}

type providerError struct {
	message    string
	statusCode int
	retryable  bool
	err        error
}

func (e providerError) Error() string { return e.message }

func (e providerError) Unwrap() error { return e.err }

func isRetryable(err error) bool {
	var providerErr providerError
	return errors.As(err, &providerErr) && providerErr.retryable
}

func (p Provider) retryDelays() []Duration {
	if p.RetryDelays != nil {
		return p.RetryDelays
	}
	return []Duration{time.Second, 2 * time.Second, 4 * time.Second}
}

func (p Provider) sleep(delay Duration) {
	if p.Sleep != nil {
		p.Sleep(delay)
		return
	}
	time.Sleep(delay)
}

func (p Provider) getenv(key string) string {
	if p.Getenv != nil {
		return p.Getenv(key)
	}
	return os.Getenv(key)
}

func (p Provider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return http.DefaultClient
}

func (p Provider) url() string {
	if p.URL != "" {
		return p.URL
	}
	return "https://api.x.ai/v1/stt"
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
