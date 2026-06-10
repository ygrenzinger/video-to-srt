package voxtral

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderTranscribesTimestampedSegmentsToSubtitleCues(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "Example [abc123].mp3")
	outputPath := filepath.Join(dir, "Example [abc123].voxtral.srt")
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
		if got := r.MultipartForm.Value["model"]; len(got) != 1 || got[0] != "voxtral-mini-latest" {
			t.Fatalf("model field = %v", got)
		}
		if got := r.MultipartForm.Value["timestamp_granularities"]; len(got) != 1 || got[0] != "segment" {
			t.Fatalf("timestamp_granularities field = %v", got)
		}
		fileHeaders := r.MultipartForm.File["file"]
		if len(fileHeaders) != 1 {
			t.Fatalf("file parts = %d", len(fileHeaders))
		}
		sawFile = true
		writeJSON(w, `{"segments":[{"start":1.25,"end":3.5,"text":"Hello"},{"start":4,"end":5.005,"text":"World"}]}`)
	}))
	defer server.Close()

	provider := Provider{
		URL:    server.URL,
		Client: server.Client(),
		Getenv: func(key string) string {
			if key == "MISTRAL_API_KEY" {
				return "test-key"
			}
			return ""
		},
	}

	got, err := provider.Transcribe(context.Background(), audioPath, outputPath, "voxtral-mini-latest")

	if err != nil {
		t.Fatalf("Transcribe() err = %v", err)
	}
	if !sawFile {
		t.Fatal("server did not receive audio file")
	}
	if len(got) != 2 || got[0].StartMS != 1250 || got[0].EndMS != 3500 || got[0].Text != "Hello" || got[1].StartMS != 4000 || got[1].EndMS != 5005 || got[1].Text != "World" {
		t.Fatalf("cues = %#v", got)
	}
}

func TestProviderFailsWithoutAPIKeyBeforeRequest(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	_, err := (Provider{URL: server.URL, Client: server.Client(), Getenv: func(string) string { return "" }}).Transcribe(context.Background(), "missing.mp3", "out.srt", "")

	if err == nil || !strings.Contains(err.Error(), "MISTRAL_API_KEY") {
		t.Fatalf("Transcribe() err = %v", err)
	}
	if called {
		t.Fatal("provider made request without API key")
	}
}

func TestProviderRejectsMalformedJSONAndMissingTimestampSegments(t *testing.T) {
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
		{name: "missing segments", body: `{"text":"plain transcript"}`, want: "no usable timestamped segments"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, tt.body)
			}))
			defer server.Close()

			_, err := (Provider{URL: server.URL, Client: server.Client(), Getenv: func(string) string { return "test-key" }}).Transcribe(context.Background(), audioPath, outputPath, "")

			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Transcribe() err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestResponseSubtitleCuesRejectsNoUsableTimestampedSegments(t *testing.T) {
	_, err := (response{Segments: []segment{
		{Start: 0, End: 1, Text: " "},
		{Start: 2, End: 2, Text: "bad timing"},
	}}).subtitleCues()

	if err == nil || !strings.Contains(err.Error(), "no usable timestamped segments") {
		t.Fatalf("subtitleCues() err = %v", err)
	}
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}
