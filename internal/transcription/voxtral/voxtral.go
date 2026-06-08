package voxtral

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"video-to-srt/internal/subtitles"
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

func (p Provider) Transcribe(ctx context.Context, audioPath, outputPath, model string) error {
	apiKey := p.getenv("MISTRAL_API_KEY")
	if apiKey == "" {
		return errors.New("missing required environment variable: MISTRAL_API_KEY")
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
		if err := multipartWriter.WriteField("timestamp_granularities", "segment"); err != nil {
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
		return providerError{message: "voxtral transcription failed: " + err.Error(), retryable: true, err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return providerError{message: fmt.Sprintf("voxtral transcription failed: HTTP %d", resp.StatusCode), statusCode: resp.StatusCode, retryable: resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500}
	}
	var result response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("voxtral transcription response was not JSON: %w", err)
	}
	cues, err := result.subtitleCues()
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
	return "https://api.mistral.ai/v1/audio/transcriptions"
}

type response struct {
	Segments []segment `json:"segments"`
}

type segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
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
