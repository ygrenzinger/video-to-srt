package transport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"video-to-srt/internal/subtitles"
)

func TestTranscribeUploadsAudioAndReturnsCues(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "audio.mp3")
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
		if got := r.MultipartForm.Value["model"]; len(got) != 1 || got[0] != "default-model" {
			t.Fatalf("model field = %v", got)
		}
		if got := r.MultipartForm.Value["timestamp_granularities"]; len(got) != 1 || got[0] != "segment" {
			t.Fatalf("timestamp_granularities field = %v", got)
		}
		fileHeaders := r.MultipartForm.File["file"]
		if len(fileHeaders) != 1 || fileHeaders[0].Filename != "audio.mp3" {
			t.Fatalf("file parts = %#v", fileHeaders)
		}
		sawFile = true
		writeJSON(w, `{"ok":true}`)
	}))
	defer server.Close()

	got, err := Transcribe(context.Background(), Request{
		ProviderName: "test",
		URL:          server.URL,
		APIKeyEnv:    "TEST_API_KEY",
		DefaultModel: "default-model",
		AudioPath:    audioPath,
		FormFields:   []FormField{{Name: "timestamp_granularities", Value: "segment"}},
		DecodeCues: func(io.Reader) ([]subtitles.Cue, error) {
			return []subtitles.Cue{{StartMS: 1000, EndMS: 2000, Text: "ok"}}, nil
		},
		Client: server.Client(),
		Getenv: func(key string) string {
			if key == "TEST_API_KEY" {
				return "test-key"
			}
			return ""
		},
	})

	if err != nil {
		t.Fatalf("Transcribe() err = %v", err)
	}
	if !sawFile {
		t.Fatal("server did not receive audio file")
	}
	if len(got) != 1 || got[0].StartMS != 1000 || got[0].EndMS != 2000 || got[0].Text != "ok" {
		t.Fatalf("cues = %#v", got)
	}
}

func TestTranscribeUsesDefaultURLAndCustomModel(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "audio.mp3")
	if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}
	var sawRequest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRequest = true
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() err = %v", err)
		}
		if got := r.MultipartForm.Value["model"]; len(got) != 1 || got[0] != "custom-model" {
			t.Fatalf("model field = %v", got)
		}
		writeJSON(w, `{}`)
	}))
	defer server.Close()

	_, err := Transcribe(context.Background(), Request{
		ProviderName: "test",
		DefaultURL:   server.URL,
		APIKeyEnv:    "TEST_API_KEY",
		Model:        "custom-model",
		DefaultModel: "default-model",
		AudioPath:    audioPath,
		DecodeCues: func(io.Reader) ([]subtitles.Cue, error) {
			return []subtitles.Cue{{StartMS: 0, EndMS: 1000, Text: "ok"}}, nil
		},
		Client: server.Client(),
		Getenv: func(string) string { return "test-key" },
	})

	if err != nil {
		t.Fatalf("Transcribe() err = %v", err)
	}
	if !sawRequest {
		t.Fatal("server did not receive request")
	}
}

func TestTranscribeFailsWithoutAPIKeyBeforeRequest(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	_, err := Transcribe(context.Background(), Request{
		ProviderName: "test",
		URL:          server.URL,
		APIKeyEnv:    "TEST_API_KEY",
		Client:       server.Client(),
		Getenv:       func(string) string { return "" },
	})

	if err == nil || !strings.Contains(err.Error(), "TEST_API_KEY") {
		t.Fatalf("Transcribe() err = %v", err)
	}
	if called {
		t.Fatal("provider made request without API key")
	}
}

func TestTranscribeRetriesTransientFailures(t *testing.T) {
	tests := []struct {
		name      string
		firstCode int
		wantTries int
		wantErr   string
	}{
		{name: "http 500", firstCode: http.StatusInternalServerError, wantTries: 2},
		{name: "http 429", firstCode: http.StatusTooManyRequests, wantTries: 2},
		{name: "http 400", firstCode: http.StatusBadRequest, wantTries: 1, wantErr: "HTTP 400"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			audioPath := filepath.Join(dir, "audio.mp3")
			if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
				t.Fatal(err)
			}
			attempts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				if attempts == 1 {
					http.Error(w, "try later", tt.firstCode)
					return
				}
				writeJSON(w, `{}`)
			}))
			defer server.Close()

			_, err := Transcribe(context.Background(), Request{
				ProviderName: "test",
				URL:          server.URL,
				APIKeyEnv:    "TEST_API_KEY",
				DefaultModel: "default-model",
				AudioPath:    audioPath,
				DecodeCues: func(io.Reader) ([]subtitles.Cue, error) {
					return []subtitles.Cue{{StartMS: 0, EndMS: 1000, Text: "ok"}}, nil
				},
				Client:      server.Client(),
				Getenv:      func(string) string { return "test-key" },
				RetryDelays: []time.Duration{0},
				Sleep:       func(time.Duration) {},
			})

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Transcribe() err = %v, want containing %q", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("Transcribe() err = %v", err)
			}
			if attempts != tt.wantTries {
				t.Fatalf("attempts = %d, want %d", attempts, tt.wantTries)
			}
		})
	}
}

func TestTranscribeRetriesNetworkFailures(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "audio.mp3")
	if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}
	transport := &flakyTransport{}

	_, err := Transcribe(context.Background(), Request{
		ProviderName: "test",
		URL:          "https://example.test/transcriptions",
		APIKeyEnv:    "TEST_API_KEY",
		DefaultModel: "default-model",
		AudioPath:    audioPath,
		DecodeCues: func(io.Reader) ([]subtitles.Cue, error) {
			return []subtitles.Cue{{StartMS: 0, EndMS: 1000, Text: "ok"}}, nil
		},
		Client:      &http.Client{Transport: transport},
		Getenv:      func(string) string { return "test-key" },
		RetryDelays: []time.Duration{0},
		Sleep:       func(time.Duration) {},
	})

	if err != nil {
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
		Body:       io.NopCloser(strings.NewReader(`{}`)),
		Request:    req,
	}, nil
}

func TestTranscribeReturnsDecodeErrors(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "audio.mp3")
	if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"bad":true}`)
	}))
	defer server.Close()

	_, err := Transcribe(context.Background(), Request{
		ProviderName: "test",
		URL:          server.URL,
		APIKeyEnv:    "TEST_API_KEY",
		DefaultModel: "default-model",
		AudioPath:    audioPath,
		DecodeCues: func(reader io.Reader) ([]subtitles.Cue, error) {
			var result struct {
				OK bool `json:"ok"`
			}
			if err := json.NewDecoder(reader).Decode(&result); err != nil {
				return nil, err
			}
			return nil, errors.New("no cues")
		},
		Client: server.Client(),
		Getenv: func(string) string { return "test-key" },
	})

	if err == nil || !strings.Contains(err.Error(), "no cues") {
		t.Fatalf("Transcribe() err = %v", err)
	}
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}
