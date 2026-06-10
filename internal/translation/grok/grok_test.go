package grok

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"video-to-srt/internal/subtitles"
)

func TestProviderTranslatesSubtitleCues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("request JSON err = %v", err)
		}
		if body["model"] != "grok-4.3" {
			t.Fatalf("model = %v", body["model"])
		}
		writeJSON(w, `{"choices":[{"message":{"content":"{\"cues\":[{\"id\":1,\"text\":\"Bonjour\"}]}"}}]}`)
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

	got, err := provider.Translate(context.Background(), "fr", []subtitles.Cue{{StartMS: 1000, EndMS: 2000, Text: "Hello"}}, "")

	if err != nil {
		t.Fatalf("Translate() err = %v", err)
	}
	if len(got) != 1 || got[0].StartMS != 1000 || got[0].EndMS != 2000 || got[0].Text != "Bonjour" {
		t.Fatalf("translated cues = %#v", got)
	}
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}
