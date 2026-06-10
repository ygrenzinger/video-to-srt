package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
		Client:      server.Client(),
		Getenv:      func(string) string { return "test-key" },
		RetryDelays: []time.Duration{},
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
		Client:      server.Client(),
		Getenv:      func(string) string { return "test-key" },
		RetryDelays: []time.Duration{},
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

func TestTranslateRetriesInvalidProviderResponses(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			writeTranslationJSON(w, `{"choices":[{"message":{"content":"{\"cues\":[{\"id\":1,\"text\":\"\"}]}"}}]}`)
			return
		}
		writeTranslationJSON(w, `{"choices":[{"message":{"content":"{\"cues\":[{\"id\":1,\"text\":\"Bonjour\"}]}"}}]}`)
	}))
	defer server.Close()

	got, err := Translate(context.Background(), Request{
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
	if len(got) != 1 || got[0].Text != "Bonjour" {
		t.Fatalf("translated cues = %#v", got)
	}
}

func TestTranslateBatchesLargeCueSets(t *testing.T) {
	requestSizes := []int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body chatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("request JSON err = %v", err)
		}
		cues := requestCues(t, body)
		requestSizes = append(requestSizes, len(cues))
		response := cueResponse{Cues: make([]translatedCue, 0, len(cues))}
		for _, cue := range cues {
			response.Cues = append(response.Cues, translatedCue{ID: cue.ID, Text: "fr " + strconv.Itoa(cue.ID)})
		}
		content, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		chat, err := json.Marshal(chatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: string(content)}},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		writeTranslationJSON(w, string(chat))
	}))
	defer server.Close()

	source := make([]subtitles.Cue, maxCuesPerTranslationRequest+3)
	for i := range source {
		source[i] = subtitles.Cue{StartMS: i * 1000, EndMS: i*1000 + 500, Text: "cue " + strconv.Itoa(i+1)}
	}

	got, err := Translate(context.Background(), Request{
		ProviderName: "test",
		URL:          server.URL,
		APIKeyEnv:    "TEST_API_KEY",
		DefaultModel: "default-model",
		Target:       "fr",
		Cues:         source,
		Client:       server.Client(),
		Getenv:       func(string) string { return "test-key" },
	})

	if err != nil {
		t.Fatalf("Translate() err = %v", err)
	}
	if len(requestSizes) != 2 || requestSizes[0] != maxCuesPerTranslationRequest || requestSizes[1] != 3 {
		t.Fatalf("request sizes = %#v", requestSizes)
	}
	if len(got) != len(source) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(source))
	}
	if got[0].Text != "fr 1" || got[maxCuesPerTranslationRequest].Text != "fr 1" {
		t.Fatalf("translated cues were not merged from per-batch IDs: %#v", got[maxCuesPerTranslationRequest])
	}
	if got[maxCuesPerTranslationRequest].StartMS != source[maxCuesPerTranslationRequest].StartMS {
		t.Fatalf("timing was not preserved: %#v", got[maxCuesPerTranslationRequest])
	}
}

func requestCues(t *testing.T, body chatRequest) []translatedCue {
	t.Helper()
	if len(body.Messages) < 2 {
		t.Fatalf("messages = %#v", body.Messages)
	}
	content := body.Messages[1].Content
	payloadStart := strings.LastIndex(content, ": ")
	if payloadStart == -1 {
		t.Fatalf("message content has no payload: %q", content)
	}
	var cues []translatedCue
	if err := json.Unmarshal([]byte(content[payloadStart+2:]), &cues); err != nil {
		t.Fatalf("cue payload JSON err = %v", err)
	}
	return cues
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
