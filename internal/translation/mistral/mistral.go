package mistral

import (
	"context"
	"net/http"
	"time"

	"video-to-srt/internal/subtitles"
	"video-to-srt/internal/translation/transport"
)

const DefaultModel = "mistral-large-latest"

type Duration = time.Duration

type Provider struct {
	URL         string
	Client      *http.Client
	Getenv      func(string) string
	RetryDelays []Duration
	Sleep       func(Duration)
}

func (p Provider) Translate(ctx context.Context, targetLanguage string, cues []subtitles.Cue, model string) ([]subtitles.Cue, error) {
	return transport.Translate(ctx, transport.Request{
		ProviderName: "mistral",
		URL:          p.URL,
		DefaultURL:   "https://api.mistral.ai/v1/chat/completions",
		APIKeyEnv:    "MISTRAL_API_KEY",
		Model:        model,
		DefaultModel: DefaultModel,
		Target:       targetLanguage,
		Cues:         cues,
		Client:       p.Client,
		Getenv:       p.Getenv,
		RetryDelays:  p.RetryDelays,
		Sleep:        p.Sleep,
	})
}
