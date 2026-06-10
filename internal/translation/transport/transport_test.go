package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"video-to-srt/internal/subtitles"
)

func TestTranslateRejectsMissingCue(t *testing.T) {
	server := translationServer(`{"choices":[{"message":{"content":"{\"cues\":[{\"id\":1,\"text\":\"Bonjour\"}]}"}}]}`)
	defer server.Close()

	_, err := Translate(context.Background(), Request{
		ProviderName: "test",
		URL:          server.URL,
		APIKeyEnv:    "TEST_API_KEY",
		DefaultModel: "default-model",
		Target:       "fr",
		Cues: []subtitles.Cue{
			{StartMS: 1000, EndMS: 2000, Text: "Hello"},
			{StartMS: 3000, EndMS: 4000, Text: "Goodbye"},
		},
		Client: server.Client(),
		Getenv: func(string) string { return "test-key" },
	})

	if err == nil || !strings.Contains(err.Error(), "returned 1 cues, want 2") {
		t.Fatalf("Translate() err = %v", err)
	}
}

func TestTranslateRejectsDuplicateCue(t *testing.T) {
	server := translationServer(`{"choices":[{"message":{"content":"{\"cues\":[{\"id\":1,\"text\":\"Bonjour\"},{\"id\":1,\"text\":\"Salut\"}]}"}}]}`)
	defer server.Close()

	_, err := Translate(context.Background(), Request{
		ProviderName: "test",
		URL:          server.URL,
		APIKeyEnv:    "TEST_API_KEY",
		DefaultModel: "default-model",
		Target:       "fr",
		Cues: []subtitles.Cue{
			{StartMS: 1000, EndMS: 2000, Text: "Hello"},
			{StartMS: 3000, EndMS: 4000, Text: "Goodbye"},
		},
		Client: server.Client(),
		Getenv: func(string) string { return "test-key" },
	})

	if err == nil || !strings.Contains(err.Error(), "duplicate cue 1") {
		t.Fatalf("Translate() err = %v", err)
	}
}

func TestTranslateRetriesTransientFailures(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "try later", http.StatusTooManyRequests)
			return
		}
		writeTranslationJSON(w, `{"choices":[{"message":{"content":"{\"cues\":[{\"id\":1,\"text\":\"Bonjour\"}]}"}}]}`)
	}))
	defer server.Close()

	_, err := Translate(context.Background(), Request{
		ProviderName: "test",
		URL:          server.URL,
		APIKeyEnv:    "TEST_API_KEY",
		DefaultModel: "default-model",
		Target:       "fr",
		Cues:         []subtitles.Cue{{StartMS: 1000, EndMS: 2000, Text: "Hello"}},
		Client:       server.Client(),
		Getenv:       func(string) string { return "test-key" },
		RetryDelays:  []time.Duration{0},
		Sleep:        func(time.Duration) {},
	})

	if err != nil {
		t.Fatalf("Translate() err = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func translationServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTranslationJSON(w, body)
	}))
}

func writeTranslationJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}
