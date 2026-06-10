package mistral

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"video-to-srt/internal/subtitles"
)

func TestProviderTranslatesSubtitleCues(t *testing.T) {
	var sawRequest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRequest = true
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
		if body["model"] != "mistral-large-latest" {
			t.Fatalf("model = %v", body["model"])
		}
		if responseFormat, ok := body["response_format"].(map[string]any); !ok || responseFormat["type"] != "json_schema" {
			t.Fatalf("response_format = %#v", body["response_format"])
		}
		if !strings.Contains(jsonString(t, body["messages"]), "French") {
			t.Fatalf("messages do not name target language: %#v", body["messages"])
		}
		writeJSON(w, `{"choices":[{"message":{"content":"{\"cues\":[{\"id\":1,\"text\":\"Bonjour\"}]}"}}]}`)
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

	got, err := provider.Translate(context.Background(), "fr", []subtitles.Cue{{StartMS: 1000, EndMS: 2000, Text: "Hello"}}, "")

	if err != nil {
		t.Fatalf("Translate() err = %v", err)
	}
	if !sawRequest {
		t.Fatal("server did not receive request")
	}
	if len(got) != 1 || got[0].StartMS != 1000 || got[0].EndMS != 2000 || got[0].Text != "Bonjour" {
		t.Fatalf("translated cues = %#v", got)
	}
}

func jsonString(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}
