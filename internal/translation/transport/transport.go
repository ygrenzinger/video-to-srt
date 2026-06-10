package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"video-to-srt/internal/subtitles"
)

type Request struct {
	ProviderName string
	URL          string
	DefaultURL   string
	APIKeyEnv    string
	Model        string
	DefaultModel string
	Target       string
	Cues         []subtitles.Cue

	Client      *http.Client
	Getenv      func(string) string
	RetryDelays []time.Duration
	Sleep       func(time.Duration)
}

const maxCuesPerTranslationRequest = 100

func Translate(ctx context.Context, req Request) ([]subtitles.Cue, error) {
	apiKey := getenv(req)(req.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("missing required environment variable: %s", req.APIKeyEnv)
	}
	model := req.Model
	if model == "" {
		model = req.DefaultModel
	}
	out := make([]subtitles.Cue, 0, len(req.Cues))
	for start := 0; start < len(req.Cues); start += maxCuesPerTranslationRequest {
		end := start + maxCuesPerTranslationRequest
		if end > len(req.Cues) {
			end = len(req.Cues)
		}
		batchReq := req
		batchReq.Cues = req.Cues[start:end]
		cues, err := translateBatch(ctx, batchReq, model, apiKey)
		if err != nil {
			return nil, fmt.Errorf("translate cues %d-%d: %w", start+1, end, err)
		}
		out = append(out, cues...)
	}
	return out, nil
}

func translateBatch(ctx context.Context, req Request, model, apiKey string) ([]subtitles.Cue, error) {
	var lastErr error
	delays := retryDelays(req)
	for attempt := 0; attempt <= len(delays); attempt++ {
		cues, err := translateOnce(ctx, req, model, apiKey)
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

func translateOnce(ctx context.Context, req Request, model, apiKey string) ([]subtitles.Cue, error) {
	body, err := json.Marshal(chatRequest{
		Model:       model,
		Messages:    messages(req.Target, req.Cues),
		Temperature: 0,
		ResponseFormat: responseFormat{
			Type: "json_schema",
			JSONSchema: jsonSchema{
				Name:   "translated_subtitle_cues",
				Strict: true,
				Schema: cueResponseSchema(),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url(req), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client(req).Do(httpReq)
	if err != nil {
		return nil, providerError{message: req.ProviderName + " translation failed: " + err.Error(), retryable: true, err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, providerError{message: fmt.Sprintf("%s translation failed: HTTP %d", req.ProviderName, resp.StatusCode), retryable: resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500}
	}
	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%s translation response was not JSON: %w", req.ProviderName, err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("%s translation response had no choices", req.ProviderName)
	}
	var translated cueResponse
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &translated); err != nil {
		return nil, fmt.Errorf("%s translation content was not JSON: %w", req.ProviderName, err)
	}
	cues, err := mergeTranslatedCues(req.Cues, translated.Cues)
	if err != nil {
		return nil, providerError{message: fmt.Sprintf("%s translation response failed validation: %s", req.ProviderName, err), retryable: true, err: err}
	}
	return cues, nil
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []message      `json:"messages"`
	Temperature    int            `json:"temperature"`
	ResponseFormat responseFormat `json:"response_format"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type       string     `json:"type"`
	JSONSchema jsonSchema `json:"json_schema"`
}

type jsonSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type cueResponse struct {
	Cues []translatedCue `json:"cues"`
}

type translatedCue struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

func messages(target string, cues []subtitles.Cue) []message {
	payload := []translatedCue{}
	for i, cue := range cues {
		payload = append(payload, translatedCue{ID: i + 1, Text: cue.Text})
	}
	data, _ := json.Marshal(payload)
	targetName := targetLanguageName(target)
	return []message{
		{Role: "system", Content: "Translate subtitle cue text. Preserve cue IDs exactly. Return only JSON matching the schema."},
		{Role: "user", Content: fmt.Sprintf("Translate these subtitle cues to %s (%s). Keep the same number of cues and IDs: %s", targetName, target, string(data))},
	}
}

func mergeTranslatedCues(source []subtitles.Cue, translated []translatedCue) ([]subtitles.Cue, error) {
	if len(translated) != len(source) {
		return nil, fmt.Errorf("translation returned %d cues, want %d", len(translated), len(source))
	}
	byID := map[int]string{}
	for _, cue := range translated {
		text := strings.TrimSpace(cue.Text)
		if text == "" {
			return nil, fmt.Errorf("translation returned empty text for cue %d", cue.ID)
		}
		if _, exists := byID[cue.ID]; exists {
			return nil, fmt.Errorf("translation returned duplicate cue %d", cue.ID)
		}
		byID[cue.ID] = text
	}
	out := make([]subtitles.Cue, len(source))
	for i, cue := range source {
		text, ok := byID[i+1]
		if !ok {
			return nil, fmt.Errorf("translation did not return cue %d", i+1)
		}
		out[i] = subtitles.Cue{StartMS: cue.StartMS, EndMS: cue.EndMS, Text: text}
	}
	return out, nil
}

func cueResponseSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"cues": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":   map[string]any{"type": "integer"},
						"text": map[string]any{"type": "string"},
					},
					"required":             []string{"id", "text"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"cues"},
		"additionalProperties": false,
	}
}

func targetLanguageName(code string) string {
	switch code {
	case "fr":
		return "French"
	case "en":
		return "English"
	case "de":
		return "German"
	default:
		return code
	}
}

type providerError struct {
	message   string
	retryable bool
	err       error
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
