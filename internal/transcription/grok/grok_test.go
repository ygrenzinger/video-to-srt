package grok

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderTranscribesWordTimestampsToSubtitleCues(t *testing.T) {
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

	got, err := provider.Transcribe(context.Background(), audioPath, outputPath, "")

	if err != nil {
		t.Fatalf("Transcribe() err = %v", err)
	}
	if !sawFile {
		t.Fatal("server did not receive audio file")
	}
	if len(got) != 1 || got[0].StartMS != 1250 || got[0].EndMS != 2480 || got[0].Text != "Hello world." {
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

	got, err := (Provider{URL: server.URL, Client: server.Client(), Getenv: func(string) string { return "test-key" }}).Transcribe(context.Background(), audioPath, outputPath, "")

	if err != nil {
		t.Fatalf("Transcribe() err = %v", err)
	}
	if len(got) != 1 || got[0].StartMS != 500 || got[0].EndMS != 1250 || got[0].Text != "Bonjour" {
		t.Fatalf("cues = %#v", got)
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

			_, err := (Provider{URL: server.URL, Client: server.Client(), Getenv: func(string) string { return "test-key" }}).Transcribe(context.Background(), audioPath, outputPath, "")

			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Transcribe() err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestResultToSubtitleCuesRejectsInvalidSegmentTimestamp(t *testing.T) {
	_, err := resultToSubtitleCues(map[string]any{
		"segments": []any{map[string]any{
			"start": "not-a-time",
			"end":   1,
			"text":  "Bonjour",
		}},
	})

	if err == nil || !strings.Contains(err.Error(), "invalid timestamp") {
		t.Fatalf("resultToSubtitleCues() err = %v", err)
	}
}

func TestWordsToSubtitleCuesSplitsLongCues(t *testing.T) {
	cues := wordsToSubtitleCues([]map[string]any{
		{"word": "One", "start": 0.0, "end": 1.0},
		{"word": "sentence.", "start": 1.0, "end": 2.0},
		{"word": "Two", "start": 2.0, "end": 3.0},
	})

	if len(cues) != 2 {
		t.Fatalf("cues = %#v, want 2 cues", cues)
	}
	if cues[0].Text != "One sentence." || cues[1].Text != "Two" {
		t.Fatalf("cue text = %#v", cues)
	}
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}
