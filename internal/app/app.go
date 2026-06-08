package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"video-to-srt/internal/source"
	"video-to-srt/internal/transcription/grok"
	"video-to-srt/internal/transcription/voxtral"
)

type SourceRequest struct {
	URL                string
	OutputDir          string
	Cookies            string
	CookiesFromBrowser string
}

type Runner struct {
	DownloadAudio func(context.Context, SourceRequest) (string, error)
	Transcribe    func(context.Context, TranscriptionRequest) error
}

type Streams struct {
	Stdout io.Writer
	Stderr io.Writer
}

type TranscriptionRequest struct {
	Provider   string
	Model      string
	AudioPath  string
	OutputPath string
}

func Run(ctx context.Context, argv []string, streams Streams, runner Runner) int {
	stdout := streams.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := streams.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	fs := flag.NewFlagSet("video-to-srt", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outputDir := fs.String("output-dir", ".", "directory for generated files")
	cookies := fs.String("youtube-cookies", "", "cookies.txt file to pass to yt-dlp")
	cookiesFromBrowser := fs.String("youtube-cookies-from-browser", "", "browser cookie store to pass to yt-dlp")
	provider := fs.String("provider", "voxtral", "transcription provider")
	model := fs.String("model", "", "transcription model")
	quiet := fs.Bool("quiet", false, "only print the final SRT path")
	if err := fs.Parse(argv); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "Error: expected exactly one YouTube URL")
		return 1
	}
	if !isYouTubeURL(fs.Arg(0)) {
		fmt.Fprintln(stderr, "Error: expected a YouTube URL")
		return 1
	}
	if *provider != "voxtral" && *provider != "grok" {
		fmt.Fprintf(stderr, "Error: unsupported provider %q\n", *provider)
		return 1
	}
	downloadAudio := runner.DownloadAudio
	if downloadAudio == nil {
		downloadAudio = func(ctx context.Context, req SourceRequest) (string, error) {
			return source.DownloadAudio(ctx, source.Request{URL: req.URL, OutputDir: req.OutputDir, Cookies: req.Cookies, CookiesFromBrowser: req.CookiesFromBrowser}, nil)
		}
	}
	transcribe := runner.Transcribe
	if transcribe == nil {
		transcribe = func(ctx context.Context, req TranscriptionRequest) error {
			switch req.Provider {
			case "voxtral":
				return voxtral.Provider{}.Transcribe(ctx, req.AudioPath, req.OutputPath, req.Model)
			case "grok":
				return grok.Provider{}.Transcribe(ctx, req.AudioPath, req.OutputPath, req.Model)
			default:
				return fmt.Errorf("unsupported provider %q", req.Provider)
			}
		}
	}
	if !*quiet {
		fmt.Fprintln(stderr, "Downloading Audio Artifact...")
	}
	audioPath, err := downloadAudio(ctx, SourceRequest{URL: fs.Arg(0), OutputDir: *outputDir, Cookies: *cookies, CookiesFromBrowser: *cookiesFromBrowser})
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if !*quiet {
		fmt.Fprintf(stderr, "Transcribing Audio Artifact with %s...\n", providerDisplayName(*provider))
	}
	outputPath := srtPath(audioPath, *provider)
	if err := transcribe(ctx, TranscriptionRequest{Provider: *provider, Model: *model, AudioPath: audioPath, OutputPath: outputPath}); err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if *quiet {
		fmt.Fprintln(stdout, outputPath)
	} else {
		fmt.Fprintln(stderr, "Output:", outputPath)
	}
	return 0
}

func providerDisplayName(provider string) string {
	switch provider {
	case "grok":
		return "Grok"
	case "voxtral":
		return "Voxtral"
	default:
		return provider
	}
}

func srtPath(audioPath, provider string) string {
	ext := filepath.Ext(audioPath)
	return strings.TrimSuffix(audioPath, ext) + "." + provider + ".srt"
}

func isYouTubeURL(input string) bool {
	parsed, err := url.Parse(input)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	switch host {
	case "youtube.com", "www.youtube.com", "m.youtube.com", "youtu.be":
		return true
	default:
		return false
	}
}
