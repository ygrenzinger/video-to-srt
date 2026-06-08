package grok

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderTranscribesWordTimestampsToSRT(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "Example [abc123].mp3")
	outputPath := filepath.Join(dir, "Example [abc123].grok.srt")
	if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}

	var sawFile bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() err = %v", err)
		}
		if got := r.MultipartForm.Value["model"]; len(got) != 1 || got[0] != "grok-transcribe-1" {
			t.Fatalf("model field = %v", got)
		}
		if got := r.MultipartForm.Value["response_format"]; len(got) != 1 || got[0] != "verbose_json" {
			t.Fatalf("response_format field = %v", got)
		}
		if got := r.MultipartForm.Value["timestamp_granularities[]"]; len(got) != 1 || got[0] != "word" {
			t.Fatalf("timestamp_granularities[] field = %v", got)
		}
		fileHeaders := r.MultipartForm.File["file"]
		if len(fileHeaders) != 1 {
			t.Fatalf("file parts = %d", len(fileHeaders))
		}
		sawFile = true
		writeJSON(w, `{"words":[{"word":"Hello","start":1.25,"end":1.5},{"word":"world.","start":1.5,"end":2.48}]}`)
	}))
	defer server.Close()

	provider := Provider{
		URL:    server.URL,
		Client: server.Client(),
		Getenv: func(key string) string {
			if key == "XAI_API_KEY" {
				return "test-key"
			}
			return ""
		},
	}

	err := provider.Transcribe(context.Background(), audioPath, outputPath, "")

	if err != nil {
		t.Fatalf("Transcribe() err = %v", err)
	}
	if !sawFile {
		t.Fatal("server did not receive audio file")
	}
	got, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	want := "1\n00:00:01,250 --> 00:00:02,480\nHello world.\n"
	if string(got) != want {
		t.Fatalf("SRT = %q\nwant %q", string(got), want)
	}
}

func TestProviderFailsWithoutAPIKeyBeforeRequest(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	err := (Provider{URL: server.URL, Client: server.Client(), Getenv: func(string) string { return "" }}).Transcribe(context.Background(), "missing.mp3", "out.srt", "")

	if err == nil || !strings.Contains(err.Error(), "XAI_API_KEY") {
		t.Fatalf("Transcribe() err = %v", err)
	}
	if called {
		t.Fatal("provider made request without API key")
	}
}

func TestProviderTranscribesSegmentsToPlainSRT(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "audio.mp3")
	outputPath := filepath.Join(dir, "out.srt")
	if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"segments":[{"start":0.5,"end":1.25,"text":"Bonjour","speaker_id":1}]}`)
	}))
	defer server.Close()

	err := (Provider{URL: server.URL, Client: server.Client(), Getenv: func(string) string { return "test-key" }}).Transcribe(context.Background(), audioPath, outputPath, "")

	if err != nil {
		t.Fatalf("Transcribe() err = %v", err)
	}
	got, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	want := "1\n00:00:00,500 --> 00:00:01,250\nBonjour\n"
	if string(got) != want {
		t.Fatalf("SRT = %q\nwant %q", string(got), want)
	}
}

func TestProviderRetriesTransientFailures(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "audio.mp3")
	outputPath := filepath.Join(dir, "out.srt")
	if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "try later", http.StatusInternalServerError)
			return
		}
		writeJSON(w, `{"words":[{"word":"ok","start":0,"end":1}]}`)
	}))
	defer server.Close()

	provider := Provider{URL: server.URL, Client: server.Client(), Getenv: func(string) string { return "test-key" }, RetryDelays: []Duration{0}, Sleep: func(Duration) {}}

	if err := provider.Transcribe(context.Background(), audioPath, outputPath, ""); err != nil {
		t.Fatalf("Transcribe() err = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestProviderRetriesNetworkFailures(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "audio.mp3")
	outputPath := filepath.Join(dir, "out.srt")
	if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}
	transport := &flakyTransport{}
	provider := Provider{
		URL:         "https://xai.example.test/v1/stt",
		Client:      &http.Client{Transport: transport},
		Getenv:      func(string) string { return "test-key" },
		RetryDelays: []Duration{0},
		Sleep:       func(Duration) {},
	}

	if err := provider.Transcribe(context.Background(), audioPath, outputPath, ""); err != nil {
		t.Fatalf("Transcribe() err = %v", err)
	}
	if transport.attempts != 2 {
		t.Fatalf("attempts = %d, want 2", transport.attempts)
	}
}

type flakyTransport struct {
	attempts int
}

func (t *flakyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.attempts++
	if t.attempts == 1 {
		return nil, errors.New("network down")
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"words":[{"word":"ok","start":0,"end":1}]}`)),
		Request:    req,
	}, nil
}

func TestProviderDoesNotRetryNonRetryableFailure(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "audio.mp3")
	if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	err := (Provider{URL: server.URL, Client: server.Client(), Getenv: func(string) string { return "test-key" }, RetryDelays: []Duration{0}, Sleep: func(Duration) {}}).Transcribe(context.Background(), audioPath, filepath.Join(dir, "out.srt"), "")

	if err == nil || !strings.Contains(err.Error(), "HTTP 400") {
		t.Fatalf("Transcribe() err = %v", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestProviderRejectsMalformedJSONAndMissingTimestampCues(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "audio.mp3")
	outputPath := filepath.Join(dir, "out.srt")
	if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "malformed", body: `{`, want: "not JSON"},
		{name: "missing cues", body: `{"text":"plain transcript"}`, want: "no timestamped transcription cues"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, tt.body)
			}))
			defer server.Close()

			err := (Provider{URL: server.URL, Client: server.Client(), Getenv: func(string) string { return "test-key" }}).Transcribe(context.Background(), audioPath, outputPath, "")

			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Transcribe() err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}
