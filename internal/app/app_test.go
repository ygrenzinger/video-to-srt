package app

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDownloadsYouTubeSourceToAudioArtifact(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "Example [abc123].mp3")
	srtPath := filepath.Join(dir, "Example [abc123].voxtral.srt")
	var got SourceRequest
	runner := Runner{
		DownloadAudio: func(ctx context.Context, req SourceRequest) (string, error) {
			got = req
			return audioPath, nil
		},
		Transcribe: func(ctx context.Context, req TranscriptionRequest) error {
			if req.AudioPath != audioPath {
				t.Fatalf("transcribe audio path = %q", req.AudioPath)
			}
			if req.OutputPath != srtPath {
				t.Fatalf("transcribe output path = %q, want %q", req.OutputPath, srtPath)
			}
			if req.Provider != "voxtral" {
				t.Fatalf("provider = %q", req.Provider)
			}
			return nil
		},
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--output-dir", dir, "https://youtu.be/abc123"}, Streams{Stdout: &stdout, Stderr: &stderr}, runner)

	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty default stdout", stdout.String())
	}
	if !strings.Contains(stderr.String(), srtPath) {
		t.Fatalf("stderr = %q, want final SRT path", stderr.String())
	}
	if got.URL != "https://youtu.be/abc123" {
		t.Fatalf("download URL = %q", got.URL)
	}
	if got.OutputDir != dir {
		t.Fatalf("download output dir = %q, want %q", got.OutputDir, dir)
	}
}

func TestRunQuietPrintsOnlyFinalSRTPath(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "Example [abc123].mp3")
	srtPath := filepath.Join(dir, "Example [abc123].voxtral.srt")
	var stdout, stderr bytes.Buffer

	code := Run(context.Background(), []string{"--quiet", "https://youtu.be/abc123"}, Streams{Stdout: &stdout, Stderr: &stderr}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) { return audioPath, nil },
		Transcribe:    func(context.Context, TranscriptionRequest) error { return nil },
	})

	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.String() != srtPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), srtPath+"\n")
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty quiet stderr", stderr.String())
	}
}

func TestRunPassesYouTubeCookieOptions(t *testing.T) {
	var got SourceRequest
	runner := Runner{
		DownloadAudio: func(ctx context.Context, req SourceRequest) (string, error) {
			got = req
			return "audio.mp3", nil
		},
		Transcribe: func(context.Context, TranscriptionRequest) error {
			return nil
		},
	}

	code := Run(context.Background(), []string{"--youtube-cookies", "cookies.txt", "--youtube-cookies-from-browser", "chrome", "https://www.youtube.com/watch?v=abc123"}, Streams{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, runner)

	if code != 0 {
		t.Fatalf("Run() code = %d", code)
	}
	if got.Cookies != "cookies.txt" || got.CookiesFromBrowser != "chrome" {
		t.Fatalf("cookie options = %q/%q", got.Cookies, got.CookiesFromBrowser)
	}
}

func TestRunPassesProviderAndModelToTranscriptionProvider(t *testing.T) {
	var got TranscriptionRequest
	code := Run(context.Background(), []string{"--provider", "voxtral", "--model", "custom-model", "https://youtu.be/abc123"}, Streams{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) {
			return "audio.mp3", nil
		},
		Transcribe: func(ctx context.Context, req TranscriptionRequest) error {
			got = req
			return nil
		},
	})

	if code != 0 {
		t.Fatalf("Run() code = %d", code)
	}
	if got.Provider != "voxtral" || got.Model != "custom-model" {
		t.Fatalf("transcription request provider/model = %q/%q", got.Provider, got.Model)
	}
}

func TestRunFailureDoesNotPrintMisleadingFinalPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"https://youtu.be/abc123"}, Streams{Stdout: &stdout, Stderr: &stderr}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) {
			return "audio.mp3", nil
		},
		Transcribe: func(context.Context, TranscriptionRequest) error {
			return errFake("provider failed")
		},
	})

	if code == 0 {
		t.Fatal("Run() code = 0")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want no final path", stdout.String())
	}
	if !strings.Contains(stderr.String(), "provider failed") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRejectsUnsupportedProvider(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"--provider", "other", "https://youtu.be/abc123"}, Streams{Stdout: &bytes.Buffer{}, Stderr: &stderr}, Runner{})

	if code == 0 {
		t.Fatal("Run() code = 0")
	}
	if !strings.Contains(stderr.String(), "unsupported provider") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRejectsBadArgumentsBeforeDownloading(t *testing.T) {
	tests := []struct {
		name string
		argv []string
	}{
		{name: "missing", argv: nil},
		{name: "extra", argv: []string{"https://youtu.be/abc123", "https://youtu.be/def456"}},
		{name: "local path", argv: []string{"clip.mp4"}},
		{name: "unsupported http", argv: []string{"https://example.com/clip.mp4"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var stderr bytes.Buffer
			code := Run(context.Background(), tt.argv, Streams{Stdout: &bytes.Buffer{}, Stderr: &stderr}, Runner{
				DownloadAudio: func(context.Context, SourceRequest) (string, error) {
					called = true
					return "", nil
				},
			})

			if code == 0 {
				t.Fatalf("Run() code = 0, stderr = %q", stderr.String())
			}
			if called {
				t.Fatal("downloader was called")
			}
			if stderr.String() == "" {
				t.Fatal("expected a clear error on stderr")
			}
		})
	}
}

type errFake string

func (e errFake) Error() string { return string(e) }
