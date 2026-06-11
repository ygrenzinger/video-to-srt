package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDownloadsYouTubeSourceToTemporaryAudio(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "Example [abc123].mp3")
	srtPath := filepath.Join(dir, "Example [abc123].voxtral.srt")
	var got SourceRequest
	runner := Runner{
		DownloadAudio: func(ctx context.Context, req SourceRequest) (string, error) {
			got = req
			return audioPath, nil
		},
		Transcribe: func(ctx context.Context, req TranscriptionRequest) ([]Cue, error) {
			if req.AudioPath != audioPath {
				t.Fatalf("transcribe audio path = %q", req.AudioPath)
			}
			if req.Model != "" {
				t.Fatalf("transcribe model = %q, want provider default", req.Model)
			}
			if req.OutputPath != srtPath {
				t.Fatalf("transcribe output path = %q, want %q", req.OutputPath, srtPath)
			}
			if req.Provider != "voxtral" {
				t.Fatalf("provider = %q", req.Provider)
			}
			return nil, nil
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

func TestRunExtractsLocalVideoSourceToTemporaryAudio(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "talk.final.mp4")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	audioPath := filepath.Join(dir, "talk.final.mp3")
	srtPath := filepath.Join(dir, "talk.final.voxtral.srt")
	var got LocalVideoRequest
	runner := Runner{
		ExtractLocalAudio: func(ctx context.Context, req LocalVideoRequest) (string, error) {
			got = req
			return audioPath, nil
		},
		Transcribe: func(ctx context.Context, req TranscriptionRequest) ([]Cue, error) {
			if req.AudioPath != audioPath {
				t.Fatalf("transcribe audio path = %q", req.AudioPath)
			}
			if req.OutputPath != srtPath {
				t.Fatalf("transcribe output path = %q, want %q", req.OutputPath, srtPath)
			}
			if req.Provider != "voxtral" {
				t.Fatalf("provider = %q", req.Provider)
			}
			return nil, nil
		},
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"transcribe", "--output-dir", dir, videoPath}, Streams{Stdout: &stdout, Stderr: &stderr}, runner)

	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if got.Path != videoPath {
		t.Fatalf("local video path = %q", got.Path)
	}
	if got.OutputDir != dir {
		t.Fatalf("local video output dir = %q, want %q", got.OutputDir, dir)
	}
	if !strings.Contains(stderr.String(), srtPath) {
		t.Fatalf("stderr = %q, want final SRT path", stderr.String())
	}
}

func TestRunTranscribesLocalAudioSourceDirectly(t *testing.T) {
	dir := t.TempDir()
	sourceDir := t.TempDir()
	audioPath := filepath.Join(sourceDir, "talk.final.mp3")
	if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}
	srtPath := filepath.Join(dir, "talk.final.voxtral.srt")
	var got TranscriptionRequest
	var stdout, stderr bytes.Buffer

	code := Run(context.Background(), []string{"--quiet", "--output-dir", dir, audioPath}, Streams{Stdout: &stdout, Stderr: &stderr}, Runner{
		ExtractLocalAudio: func(context.Context, LocalVideoRequest) (string, error) {
			t.Fatal("local video extractor was called")
			return "", nil
		},
		Transcribe: func(ctx context.Context, req TranscriptionRequest) ([]Cue, error) {
			got = req
			return nil, nil
		},
	})

	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if got.Provider != "voxtral" || got.AudioPath != audioPath || got.OutputPath != srtPath {
		t.Fatalf("transcription request = %#v", got)
	}
	if stdout.String() != srtPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), srtPath+"\n")
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty quiet stderr", stderr.String())
	}
	if _, err := os.Stat(audioPath); err != nil {
		t.Fatalf("local audio source was not preserved: %v", err)
	}
}

func TestRunAcceptsConfiguredLocalAudioExtensions(t *testing.T) {
	dir := t.TempDir()
	for _, ext := range []string{".mp3", ".wav", ".flac", ".ogg"} {
		t.Run(ext, func(t *testing.T) {
			audioPath := filepath.Join(dir, "clip"+ext)
			if err := os.WriteFile(audioPath, []byte("audio"), 0o644); err != nil {
				t.Fatal(err)
			}
			called := false
			code := Run(context.Background(), []string{audioPath}, Streams{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, Runner{
				Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
					called = true
					return nil, nil
				},
			})

			if code != 0 {
				t.Fatalf("Run() code = %d", code)
			}
			if !called {
				t.Fatal("transcription was not called")
			}
			if _, err := os.Stat(audioPath); err != nil {
				t.Fatalf("local audio source was not preserved: %v", err)
			}
		})
	}
}

func TestRunAcceptsConfiguredLocalVideoExtensions(t *testing.T) {
	dir := t.TempDir()
	for _, ext := range []string{".mp4", ".mov", ".mkv", ".webm", ".avi", ".m4v"} {
		t.Run(ext, func(t *testing.T) {
			videoPath := filepath.Join(dir, "clip"+ext)
			if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
				t.Fatal(err)
			}
			called := false
			code := Run(context.Background(), []string{videoPath}, Streams{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, Runner{
				ExtractLocalAudio: func(context.Context, LocalVideoRequest) (string, error) {
					called = true
					return filepath.Join(dir, "clip.mp3"), nil
				},
				Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
					return nil, nil
				},
			})

			if code != 0 {
				t.Fatalf("Run() code = %d", code)
			}
			if !called {
				t.Fatal("local extractor was not called")
			}
		})
	}
}

func TestRunQuietPrintsOnlyFinalSRTPath(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "Example [abc123].mp3")
	srtPath := filepath.Join(dir, "Example [abc123].voxtral.srt")
	var stdout, stderr bytes.Buffer

	code := Run(context.Background(), []string{"--quiet", "https://youtu.be/abc123"}, Streams{Stdout: &stdout, Stderr: &stderr}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) { return audioPath, nil },
		Transcribe:    func(context.Context, TranscriptionRequest) ([]Cue, error) { return nil, nil },
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

func TestRunPrintsVersionAndExits(t *testing.T) {
	oldVersion := Version
	Version = "v1.2.3"
	t.Cleanup(func() { Version = oldVersion })

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--version"}, Streams{Stdout: &stdout, Stderr: &stderr}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) {
			t.Fatal("downloader was called")
			return "", nil
		},
	})

	if code != 0 {
		t.Fatalf("Run() code = %d", code)
	}
	if stdout.String() != "v1.2.3\n" {
		t.Fatalf("stdout = %q, want version", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty stderr", stderr.String())
	}
}

func TestRunPrintsHelpAndExits(t *testing.T) {
	var stdout, stderr bytes.Buffer
	called := false

	code := Run(context.Background(), []string{"--help"}, Streams{Stdout: &stdout, Stderr: &stderr}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) {
			called = true
			return "", nil
		},
		ExtractLocalAudio: func(context.Context, LocalVideoRequest) (string, error) {
			called = true
			return "", nil
		},
		Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
			called = true
			return nil, nil
		},
		Translate: func(context.Context, TranslationRequest) ([]Cue, error) {
			called = true
			return nil, nil
		},
	})

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if called {
		t.Fatal("runner was called")
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty stderr", stderr.String())
	}
	help := stdout.String()
	for _, want := range []string{
		"video-to-srt [transcribe] [options] <media-source>",
		"video-to-srt translate [options] <subtitle-source.srt>",
		"YouTube Source:",
		"Local Video Source:",
		".mp4, .mov, .mkv, .webm, .avi, or .m4v",
		"Local Audio Source:",
		".mp3, .wav, .flac, or .ogg",
		"Subtitle Source:",
		"requires --target-language",
		"--provider <voxtral|grok>",
		"--translation-provider <mistral|grok>",
		"transcribe",
		"translate",
		"yt-dlp",
		"ffmpeg",
		"MISTRAL_API_KEY",
		"XAI_API_KEY",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("help output missing %q:\n%s", want, help)
		}
	}
}

func TestRunRemovesTemporaryAudioAfterSuccessfulTranscription(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "Example [abc123].mp3")
	if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}

	code := Run(context.Background(), []string{"https://youtu.be/abc123"}, Streams{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) {
			return audioPath, nil
		},
		Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
			return nil, nil
		},
	})

	if code != 0 {
		t.Fatalf("Run() code = %d", code)
	}
	if _, err := os.Stat(audioPath); !os.IsNotExist(err) {
		t.Fatalf("temporary audio exists after successful transcription: %v", err)
	}
}

func TestRunRemovesTemporaryAudioAfterFailedTranscription(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "Example [abc123].mp3")
	if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"https://youtu.be/abc123"}, Streams{Stdout: &bytes.Buffer{}, Stderr: &stderr}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) {
			return audioPath, nil
		},
		Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
			return nil, errFake("provider failed")
		},
	})

	if code == 0 {
		t.Fatal("Run() code = 0")
	}
	if _, err := os.Stat(audioPath); !os.IsNotExist(err) {
		t.Fatalf("temporary audio exists after failed transcription: %v", err)
	}
	if !strings.Contains(stderr.String(), "provider failed") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunLocalVideoSourceSupportsProviderModelAndQuiet(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "demo.mov")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	audioPath := filepath.Join(dir, "demo.mp3")
	srtPath := filepath.Join(dir, "demo.grok.srt")
	var got TranscriptionRequest
	var stdout, stderr bytes.Buffer

	code := Run(context.Background(), []string{"--quiet", "--provider", "grok", "--model", "custom-model", "--output-dir", dir, videoPath}, Streams{Stdout: &stdout, Stderr: &stderr}, Runner{
		ExtractLocalAudio: func(context.Context, LocalVideoRequest) (string, error) {
			return audioPath, nil
		},
		Transcribe: func(ctx context.Context, req TranscriptionRequest) ([]Cue, error) {
			got = req
			return nil, nil
		},
	})

	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if got.Provider != "grok" || got.Model != "custom-model" || got.AudioPath != audioPath || got.OutputPath != srtPath {
		t.Fatalf("transcription request = %#v", got)
	}
	if stdout.String() != srtPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), srtPath+"\n")
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty quiet stderr", stderr.String())
	}
}

func TestRunLocalVideoExtractionFailureDoesNotTranscribe(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "demo.mov")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	transcribed := false

	code := Run(context.Background(), []string{"--quiet", videoPath}, Streams{Stdout: &stdout, Stderr: &stderr}, Runner{
		ExtractLocalAudio: func(context.Context, LocalVideoRequest) (string, error) {
			return "", errFake("ffmpeg failed")
		},
		Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
			transcribed = true
			return nil, nil
		},
	})

	if code == 0 {
		t.Fatal("Run() code = 0")
	}
	if transcribed {
		t.Fatal("transcription was called")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want no final path", stdout.String())
	}
	if !strings.Contains(stderr.String(), "ffmpeg failed") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunPassesYouTubeCookieOptions(t *testing.T) {
	var got SourceRequest
	runner := Runner{
		DownloadAudio: func(ctx context.Context, req SourceRequest) (string, error) {
			got = req
			return "audio.mp3", nil
		},
		Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
			return nil, nil
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

func TestRunRejectsYouTubeCookieOptionsForLocalVideoSources(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"--youtube-cookies", "cookies.txt", videoPath}, Streams{Stdout: &bytes.Buffer{}, Stderr: &stderr}, Runner{
		ExtractLocalAudio: func(context.Context, LocalVideoRequest) (string, error) {
			called = true
			return "", nil
		},
	})

	if code == 0 {
		t.Fatal("Run() code = 0")
	}
	if called {
		t.Fatal("local extractor was called")
	}
	if !strings.Contains(stderr.String(), "YouTube cookie options") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRejectsYouTubeCookiesFromBrowserForLocalVideoSources(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"--youtube-cookies-from-browser", "chrome", videoPath}, Streams{Stdout: &bytes.Buffer{}, Stderr: &stderr}, Runner{
		ExtractLocalAudio: func(context.Context, LocalVideoRequest) (string, error) {
			called = true
			return "", nil
		},
		Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
			called = true
			return nil, nil
		},
	})

	if code == 0 {
		t.Fatal("Run() code = 0")
	}
	if called {
		t.Fatal("local extraction or transcription was called")
	}
	if !strings.Contains(stderr.String(), "YouTube cookie options") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRejectsYouTubeCookieOptionsForLocalAudioSources(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "clip.mp3")
	if err := os.WriteFile(audioPath, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"--youtube-cookies", "cookies.txt", audioPath}, Streams{Stdout: &bytes.Buffer{}, Stderr: &stderr}, Runner{
		Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
			called = true
			return nil, nil
		},
	})

	if code == 0 {
		t.Fatal("Run() code = 0")
	}
	if called {
		t.Fatal("transcription was called")
	}
	if !strings.Contains(stderr.String(), "YouTube cookie options") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if _, err := os.Stat(audioPath); err != nil {
		t.Fatalf("local audio source was not preserved: %v", err)
	}
}

func TestRunPassesProviderAndModelToTranscriptionProvider(t *testing.T) {
	var got TranscriptionRequest
	code := Run(context.Background(), []string{"--provider", "voxtral", "--model", "custom-model", "https://youtu.be/abc123"}, Streams{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) {
			return "audio.mp3", nil
		},
		Transcribe: func(ctx context.Context, req TranscriptionRequest) ([]Cue, error) {
			got = req
			return nil, nil
		},
	})

	if code != 0 {
		t.Fatalf("Run() code = %d", code)
	}
	if got.Provider != "voxtral" || got.Model != "custom-model" {
		t.Fatalf("transcription request provider/model = %q/%q", got.Provider, got.Model)
	}
}

func TestRunAcceptsGrokProvider(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "Example [abc123].mp3")
	srtPath := filepath.Join(dir, "Example [abc123].grok.srt")
	var got TranscriptionRequest
	code := Run(context.Background(), []string{"--provider", "grok", "https://youtu.be/abc123"}, Streams{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) {
			return audioPath, nil
		},
		Transcribe: func(ctx context.Context, req TranscriptionRequest) ([]Cue, error) {
			got = req
			return nil, nil
		},
	})

	if code != 0 {
		t.Fatalf("Run() code = %d", code)
	}
	if got.Provider != "grok" || got.Model != "" || got.OutputPath != srtPath {
		t.Fatalf("transcription request = %#v", got)
	}
}

func TestRunRejectsUnsupportedTargetLanguageBeforePreparingSource(t *testing.T) {
	called := false
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"--target-language", "fr-FR", "https://youtu.be/abc123"}, Streams{Stdout: &bytes.Buffer{}, Stderr: &stderr}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) {
			called = true
			return "", nil
		},
		Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
			called = true
			return nil, nil
		},
	})

	if code == 0 {
		t.Fatal("Run() code = 0")
	}
	if called {
		t.Fatal("source preparation or transcription was called")
	}
	if !strings.Contains(stderr.String(), "unsupported target language") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunTranslatesMediaSourceWhenTargetLanguageIsRequested(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "talk.mp3")
	sourcePath := filepath.Join(dir, "talk.voxtral.srt")
	translatedPath := filepath.Join(dir, "talk.voxtral.fr.srt")
	var gotTranscription TranscriptionRequest
	var gotTranslation TranslationRequest
	var stdout, stderr bytes.Buffer

	code := Run(context.Background(), []string{"--target-language", "fr", "--quiet", "https://youtu.be/abc123"}, Streams{Stdout: &stdout, Stderr: &stderr}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) {
			return audioPath, nil
		},
		Transcribe: func(ctx context.Context, req TranscriptionRequest) ([]Cue, error) {
			gotTranscription = req
			return []Cue{{StartMS: 1000, EndMS: 2000, Text: "Hello"}}, nil
		},
		Translate: func(ctx context.Context, req TranslationRequest) ([]Cue, error) {
			gotTranslation = req
			return []Cue{{StartMS: 1000, EndMS: 2000, Text: "Bonjour"}}, nil
		},
	})

	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if gotTranscription.OutputPath != sourcePath {
		t.Fatalf("transcription output path = %q, want %q", gotTranscription.OutputPath, sourcePath)
	}
	if gotTranslation.Provider != "mistral" || gotTranslation.Model != "" || gotTranslation.TargetLanguage != "fr" || gotTranslation.OutputPath != translatedPath {
		t.Fatalf("translation request = %#v", gotTranslation)
	}
	if len(gotTranslation.Cues) != 1 || gotTranslation.Cues[0].Text != "Hello" {
		t.Fatalf("translation cues = %#v", gotTranslation.Cues)
	}
	if stdout.String() != sourcePath+"\n"+translatedPath+"\n" {
		t.Fatalf("stdout = %q, want both SRT paths", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty quiet stderr", stderr.String())
	}
	gotSource, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gotSource), "Hello") {
		t.Fatalf("source SRT = %q", string(gotSource))
	}
	gotTranslated, err := os.ReadFile(translatedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gotTranslated), "Bonjour") {
		t.Fatalf("translated SRT = %q", string(gotTranslated))
	}
}

func TestRunKeepsSourceSRTAndPrintsNoQuietSuccessPathsWhenTranslationFails(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "talk.mp3")
	sourcePath := filepath.Join(dir, "talk.voxtral.srt")
	translatedPath := filepath.Join(dir, "talk.voxtral.fr.srt")
	var stdout, stderr bytes.Buffer

	code := Run(context.Background(), []string{"--target-language", "fr", "--quiet", "https://youtu.be/abc123"}, Streams{Stdout: &stdout, Stderr: &stderr}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) {
			return audioPath, nil
		},
		Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
			return []Cue{{StartMS: 1000, EndMS: 2000, Text: "Hello"}}, nil
		},
		Translate: func(context.Context, TranslationRequest) ([]Cue, error) {
			return nil, errFake("translation failed")
		},
	})

	if code == 0 {
		t.Fatal("Run() code = 0")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want no quiet success paths", stdout.String())
	}
	if !strings.Contains(stderr.String(), "translation failed") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if _, err := os.Stat(sourcePath); err != nil {
		t.Fatalf("source SRT was not kept: %v", err)
	}
	if _, err := os.Stat(translatedPath); !os.IsNotExist(err) {
		t.Fatalf("translated SRT exists after failed translation: %v", err)
	}
}

func TestRunTranslatesSubtitleSourceWhenTargetLanguageIsRequested(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "talk.voxtral.srt")
	translatedPath := filepath.Join(dir, "talk.voxtral.fr.srt")
	if err := os.WriteFile(sourcePath, []byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var got TranslationRequest
	var stdout, stderr bytes.Buffer

	code := Run(context.Background(), []string{"translate", "--quiet", "--target-language", "fr", sourcePath}, Streams{Stdout: &stdout, Stderr: &stderr}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) {
			t.Fatal("downloader was called")
			return "", nil
		},
		ExtractLocalAudio: func(context.Context, LocalVideoRequest) (string, error) {
			t.Fatal("local extractor was called")
			return "", nil
		},
		Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
			t.Fatal("transcription was called")
			return nil, nil
		},
		Translate: func(ctx context.Context, req TranslationRequest) ([]Cue, error) {
			got = req
			return []Cue{{StartMS: 1000, EndMS: 2000, Text: "Bonjour"}}, nil
		},
	})

	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if got.Provider != "mistral" || got.TargetLanguage != "fr" || got.OutputPath != translatedPath {
		t.Fatalf("translation request = %#v", got)
	}
	if len(got.Cues) != 1 || got.Cues[0].Text != "Hello" {
		t.Fatalf("translation cues = %#v", got.Cues)
	}
	if stdout.String() != translatedPath+"\n" {
		t.Fatalf("stdout = %q, want translated SRT path", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty quiet stderr", stderr.String())
	}
	if gotSource, err := os.ReadFile(sourcePath); err != nil || !strings.Contains(string(gotSource), "Hello") {
		t.Fatalf("source SRT changed or unreadable: %q %v", string(gotSource), err)
	}
	gotTranslated, err := os.ReadFile(translatedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gotTranslated), "Bonjour") {
		t.Fatalf("translated SRT = %q", string(gotTranslated))
	}
}

func TestRunRejectsBareSubtitleSource(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "talk.voxtral.srt")
	if err := os.WriteFile(sourcePath, []byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"--target-language", "fr", sourcePath}, Streams{Stdout: &bytes.Buffer{}, Stderr: &stderr}, Runner{
		Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
			called = true
			return nil, nil
		},
		Translate: func(context.Context, TranslationRequest) ([]Cue, error) {
			called = true
			return nil, nil
		},
	})

	if code == 0 {
		t.Fatal("Run() code = 0")
	}
	if called {
		t.Fatal("runner was called")
	}
	if !strings.Contains(stderr.String(), "local video file") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunTranslateRequiresTargetLanguage(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "talk.voxtral.srt")
	if err := os.WriteFile(sourcePath, []byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"translate", sourcePath}, Streams{Stdout: &bytes.Buffer{}, Stderr: &stderr}, Runner{
		Translate: func(context.Context, TranslationRequest) ([]Cue, error) {
			called = true
			return nil, nil
		},
	})

	if code == 0 {
		t.Fatal("Run() code = 0")
	}
	if called {
		t.Fatal("translation was called")
	}
	if !strings.Contains(stderr.String(), "--target-language") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFailureDoesNotPrintMisleadingFinalPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"https://youtu.be/abc123"}, Streams{Stdout: &stdout, Stderr: &stderr}, Runner{
		DownloadAudio: func(context.Context, SourceRequest) (string, error) {
			return "audio.mp3", nil
		},
		Transcribe: func(context.Context, TranscriptionRequest) ([]Cue, error) {
			return nil, errFake("provider failed")
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
	if !strings.Contains(stderr.String(), "--provider must be one of") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRejectsBadArgumentsBeforeDownloading(t *testing.T) {
	dir := t.TempDir()
	unsupportedPath := filepath.Join(dir, "clip.txt")
	if err := os.WriteFile(unsupportedPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	videoDir := filepath.Join(dir, "folder.mp4")
	if err := os.Mkdir(videoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		argv []string
	}{
		{name: "missing", argv: nil},
		{name: "extra", argv: []string{"https://youtu.be/abc123", "https://youtu.be/def456"}},
		{name: "missing local video", argv: []string{filepath.Join(dir, "missing.mp4")}},
		{name: "unsupported local extension", argv: []string{unsupportedPath}},
		{name: "local video directory", argv: []string{videoDir}},
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
				ExtractLocalAudio: func(context.Context, LocalVideoRequest) (string, error) {
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
