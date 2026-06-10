package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"video-to-srt/internal/subtitles"
)

type FormField struct {
	Name  string
	Value string
}

type Request struct {
	ProviderName string
	URL          string
	DefaultURL   string
	APIKeyEnv    string
	Model        string
	DefaultModel string
	AudioPath    string
	OutputPath   string
	FormFields   []FormField
	DecodeCues   func(io.Reader) ([]subtitles.Cue, error)

	Client      *http.Client
	Getenv      func(string) string
	RetryDelays []time.Duration
	Sleep       func(time.Duration)
}

func Transcribe(ctx context.Context, req Request) ([]subtitles.Cue, error) {
	apiKey := getenv(req)(req.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("missing required environment variable: %s", req.APIKeyEnv)
	}
	model := req.Model
	if model == "" {
		model = req.DefaultModel
	}
	var lastErr error
	delays := retryDelays(req)
	for attempt := 0; attempt <= len(delays); attempt++ {
		cues, err := transcribeOnce(ctx, req, model, apiKey)
		if err == nil {
			return cues, nil
		}
		lastErr = err
		if attempt == len(delays) || !isRetryable(err) {
			return nil, err
		}
		sleep(req)(delays[attempt])
	}
	return nil, lastErr
}

func transcribeOnce(ctx context.Context, req Request, model, apiKey string) ([]subtitles.Cue, error) {
	bodyReader, bodyWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(bodyWriter)
	go func() {
		defer bodyWriter.Close()
		defer multipartWriter.Close()
		if err := multipartWriter.WriteField("model", model); err != nil {
			_ = bodyWriter.CloseWithError(err)
			return
		}
		for _, field := range req.FormFields {
			if err := multipartWriter.WriteField(field.Name, field.Value); err != nil {
				_ = bodyWriter.CloseWithError(err)
				return
			}
		}
		file, err := os.Open(req.AudioPath)
		if err != nil {
			_ = bodyWriter.CloseWithError(err)
			return
		}
		defer file.Close()
		part, err := multipartWriter.CreateFormFile("file", filepath.Base(req.AudioPath))
		if err != nil {
			_ = bodyWriter.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			_ = bodyWriter.CloseWithError(err)
		}
	}()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url(req), bodyReader)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	resp, err := client(req).Do(httpReq)
	if err != nil {
		return nil, providerError{message: req.ProviderName + " transcription failed: " + err.Error(), retryable: true, err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, providerError{message: fmt.Sprintf("%s transcription failed: HTTP %d", req.ProviderName, resp.StatusCode), statusCode: resp.StatusCode, retryable: resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500}
	}
	cues, err := req.DecodeCues(resp.Body)
	if err != nil {
		return nil, err
	}
	return cues, nil
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

func retryDelays(req Request) []time.Duration {
	if req.RetryDelays != nil {
		return req.RetryDelays
	}
	return []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
}

func sleep(req Request) func(time.Duration) {
	if req.Sleep != nil {
		return req.Sleep
	}
	return time.Sleep
}

func getenv(req Request) func(string) string {
	if req.Getenv != nil {
		return req.Getenv
	}
	return os.Getenv
}

func client(req Request) *http.Client {
	if req.Client != nil {
		return req.Client
	}
	return http.DefaultClient
}

func url(req Request) string {
	if req.URL != "" {
		return req.URL
	}
	return req.DefaultURL
}
