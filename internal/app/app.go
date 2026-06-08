package app

import (
	"context"
	"errors"
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

type LocalVideoRequest struct {
	Path      string
	OutputDir string
}

type Runner struct {
	DownloadAudio     func(context.Context, SourceRequest) (string, error)
	ExtractLocalAudio func(context.Context, LocalVideoRequest) (string, error)
	Transcribe        func(context.Context, TranscriptionRequest) error
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

var Version = "dev"

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
	showVersion := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(argv); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, Version)
		return 0
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "Error: expected exactly one Media Source")
		return 1
	}
	mediaSource, err := classifyMediaSource(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if mediaSource.kind == mediaSourceLocalVideo && (*cookies != "" || *cookiesFromBrowser != "") {
		fmt.Fprintln(stderr, "Error: YouTube cookie options can only be used with YouTube Sources")
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
	extractLocalAudio := runner.ExtractLocalAudio
	if extractLocalAudio == nil {
		extractLocalAudio = func(ctx context.Context, req LocalVideoRequest) (string, error) {
			return source.ExtractLocalAudio(ctx, source.LocalRequest{Path: req.Path, OutputDir: req.OutputDir}, nil)
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
		fmt.Fprintln(stderr, "Preparing Media Source...")
	}
	var audioPath string
	switch mediaSource.kind {
	case mediaSourceYouTube:
		audioPath, err = downloadAudio(ctx, SourceRequest{URL: mediaSource.value, OutputDir: *outputDir, Cookies: *cookies, CookiesFromBrowser: *cookiesFromBrowser})
	case mediaSourceLocalVideo:
		audioPath, err = extractLocalAudio(ctx, LocalVideoRequest{Path: mediaSource.value, OutputDir: *outputDir})
	}
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if !*quiet {
		fmt.Fprintf(stderr, "Transcribing with %s...\n", providerDisplayName(*provider))
	}
	outputPath := srtPath(audioPath, *provider)
	transcribeErr := transcribe(ctx, TranscriptionRequest{Provider: *provider, Model: *model, AudioPath: audioPath, OutputPath: outputPath})
	cleanupErr := removeTemporaryAudio(audioPath)
	if transcribeErr != nil {
		fmt.Fprintln(stderr, "Error:", transcribeErr)
		return 1
	}
	if cleanupErr != nil {
		fmt.Fprintln(stderr, "Error:", cleanupErr)
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

func removeTemporaryAudio(audioPath string) error {
	if err := os.Remove(audioPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove temporary audio file: %w", err)
	}
	return nil
}

type mediaSourceKind int

const (
	mediaSourceYouTube mediaSourceKind = iota
	mediaSourceLocalVideo
)

type mediaSource struct {
	kind  mediaSourceKind
	value string
}

func classifyMediaSource(input string) (mediaSource, error) {
	if isYouTubeURL(input) {
		return mediaSource{kind: mediaSourceYouTube, value: input}, nil
	}
	parsed, err := url.Parse(input)
	if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		return mediaSource{}, errors.New("non-YouTube HTTP media sources are not supported")
	}
	if !isAcceptedLocalVideoExtension(input) {
		return mediaSource{}, fmt.Errorf("expected a YouTube URL or local video file (%s)", strings.Join(acceptedLocalVideoExtensions, ", "))
	}
	info, err := os.Stat(input)
	if err != nil {
		return mediaSource{}, fmt.Errorf("local video file is not readable: %w", err)
	}
	if info.IsDir() {
		return mediaSource{}, errors.New("local video source must be a file, not a directory")
	}
	return mediaSource{kind: mediaSourceLocalVideo, value: input}, nil
}

var acceptedLocalVideoExtensions = []string{".mp4", ".mov", ".mkv", ".webm", ".avi", ".m4v"}

func isAcceptedLocalVideoExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, accepted := range acceptedLocalVideoExtensions {
		if ext == accepted {
			return true
		}
	}
	return false
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
