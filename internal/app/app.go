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
	"video-to-srt/internal/subtitles"
	groktranscription "video-to-srt/internal/transcription/grok"
	"video-to-srt/internal/transcription/voxtral"
	groktranslation "video-to-srt/internal/translation/grok"
	mistraltranslation "video-to-srt/internal/translation/mistral"
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
	Transcribe        func(context.Context, TranscriptionRequest) ([]Cue, error)
	Translate         func(context.Context, TranslationRequest) ([]Cue, error)
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

type TranslationRequest struct {
	Provider       string
	Model          string
	TargetLanguage string
	Cues           []Cue
	OutputPath     string
}

type Cue = subtitles.Cue

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
	targetLanguage := fs.String("target-language", "", "target language for translated subtitles")
	translationProvider := fs.String("translation-provider", "", "translation provider")
	translationModel := fs.String("translation-model", "", "translation model")
	quiet := fs.Bool("quiet", false, "only print generated SRT paths")
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
	if *targetLanguage != "" && !isAcceptedTargetLanguage(*targetLanguage) {
		fmt.Fprintf(stderr, "Error: unsupported target language %q\n", *targetLanguage)
		return 1
	}
	translate := runner.Translate
	if translate == nil {
		translate = func(ctx context.Context, req TranslationRequest) ([]Cue, error) {
			switch req.Provider {
			case "mistral":
				return mistraltranslation.Provider{}.Translate(ctx, req.TargetLanguage, req.Cues, req.Model)
			case "grok":
				return groktranslation.Provider{}.Translate(ctx, req.TargetLanguage, req.Cues, req.Model)
			default:
				return nil, fmt.Errorf("unsupported translation provider %q", req.Provider)
			}
		}
	}
	if isSubtitleSourcePath(fs.Arg(0)) {
		return runSubtitleSource(ctx, fs.Arg(0), *targetLanguage, *translationProvider, *translationModel, *quiet, stdout, stderr, translate)
	}
	mediaSource, err := classifyMediaSource(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if mediaSource.kind != mediaSourceYouTube && (*cookies != "" || *cookiesFromBrowser != "") {
		fmt.Fprintln(stderr, "Error: YouTube cookie options can only be used with YouTube Sources")
		return 1
	}
	if *provider != "voxtral" && *provider != "grok" {
		fmt.Fprintf(stderr, "Error: unsupported provider %q\n", *provider)
		return 1
	}
	if *translationProvider != "" && *translationProvider != "mistral" && *translationProvider != "grok" {
		fmt.Fprintf(stderr, "Error: unsupported translation provider %q\n", *translationProvider)
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
		transcribe = func(ctx context.Context, req TranscriptionRequest) ([]Cue, error) {
			switch req.Provider {
			case "voxtral":
				return voxtral.Provider{}.Transcribe(ctx, req.AudioPath, req.OutputPath, req.Model)
			case "grok":
				return groktranscription.Provider{}.Transcribe(ctx, req.AudioPath, req.OutputPath, req.Model)
			default:
				return nil, fmt.Errorf("unsupported provider %q", req.Provider)
			}
		}
	}
	if !*quiet {
		fmt.Fprintln(stderr, "Preparing Media Source...")
	}
	var audioPath string
	cleanupAudio := true
	switch mediaSource.kind {
	case mediaSourceYouTube:
		audioPath, err = downloadAudio(ctx, SourceRequest{URL: mediaSource.value, OutputDir: *outputDir, Cookies: *cookies, CookiesFromBrowser: *cookiesFromBrowser})
	case mediaSourceLocalVideo:
		audioPath, err = extractLocalAudio(ctx, LocalVideoRequest{Path: mediaSource.value, OutputDir: *outputDir})
	case mediaSourceLocalAudio:
		audioPath = mediaSource.value
		cleanupAudio = false
	}
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if !*quiet {
		fmt.Fprintf(stderr, "Transcribing with %s...\n", providerDisplayName(*provider))
	}
	outputPath := srtPath(audioPath, *provider)
	if mediaSource.kind == mediaSourceLocalAudio {
		outputPath = outputSRTPath(*outputDir, audioPath, *provider)
	}
	cues, transcribeErr := transcribe(ctx, TranscriptionRequest{Provider: *provider, Model: *model, AudioPath: audioPath, OutputPath: outputPath})
	if transcribeErr == nil && cues != nil {
		transcribeErr = subtitles.AtomicWriteSRT(outputPath, cues)
	}
	var cleanupErr error
	if cleanupAudio {
		cleanupErr = removeTemporaryAudio(audioPath)
	}
	if transcribeErr != nil {
		fmt.Fprintln(stderr, "Error:", transcribeErr)
		return 1
	}
	if cleanupErr != nil {
		fmt.Fprintln(stderr, "Error:", cleanupErr)
		return 1
	}
	if *targetLanguage != "" {
		selectedTranslationProvider := *translationProvider
		if selectedTranslationProvider == "" {
			selectedTranslationProvider = defaultTranslationProvider(*provider)
		}
		translatedPath := translatedSRTPath(outputPath, *targetLanguage)
		translatedCues, err := translate(ctx, TranslationRequest{Provider: selectedTranslationProvider, Model: *translationModel, TargetLanguage: *targetLanguage, Cues: cues, OutputPath: translatedPath})
		if err != nil {
			fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		if err := subtitles.AtomicWriteSRT(translatedPath, translatedCues); err != nil {
			fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		if *quiet {
			fmt.Fprintln(stdout, outputPath)
			fmt.Fprintln(stdout, translatedPath)
		} else {
			fmt.Fprintln(stderr, "Output:", outputPath)
			fmt.Fprintln(stderr, "Translated output:", translatedPath)
		}
		return 0
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

func outputSRTPath(outputDir, sourcePath, provider string) string {
	if outputDir == "" {
		outputDir = "."
	}
	base := filepath.Base(sourcePath)
	ext := filepath.Ext(base)
	return filepath.Join(outputDir, strings.TrimSuffix(base, ext)+"."+provider+".srt")
}

func translatedSRTPath(sourceSRTPath, targetLanguage string) string {
	ext := filepath.Ext(sourceSRTPath)
	return strings.TrimSuffix(sourceSRTPath, ext) + "." + targetLanguage + ext
}

func defaultTranslationProvider(transcriptionProvider string) string {
	if transcriptionProvider == "grok" {
		return "grok"
	}
	return "mistral"
}

func runSubtitleSource(ctx context.Context, path, targetLanguage, translationProvider, translationModel string, quiet bool, stdout, stderr io.Writer, translate func(context.Context, TranslationRequest) ([]Cue, error)) int {
	if targetLanguage == "" {
		fmt.Fprintln(stderr, "Error: Subtitle Source requires --target-language")
		return 1
	}
	if translationProvider == "" {
		translationProvider = "mistral"
	}
	if translationProvider != "mistral" && translationProvider != "grok" {
		fmt.Fprintf(stderr, "Error: unsupported translation provider %q\n", translationProvider)
		return 1
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(stderr, "Error: subtitle source is not readable:", err)
		return 1
	}
	cues, err := subtitles.ParseSRT(string(data))
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	outputPath := translatedSRTPath(path, targetLanguage)
	translatedCues, err := translate(ctx, TranslationRequest{Provider: translationProvider, Model: translationModel, TargetLanguage: targetLanguage, Cues: cues, OutputPath: outputPath})
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if err := subtitles.AtomicWriteSRT(outputPath, translatedCues); err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if quiet {
		fmt.Fprintln(stdout, outputPath)
	} else {
		fmt.Fprintln(stderr, "Translated output:", outputPath)
	}
	return 0
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
	mediaSourceLocalAudio
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
	if !isAcceptedLocalVideoExtension(input) && !isAcceptedLocalAudioExtension(input) {
		return mediaSource{}, fmt.Errorf("expected a YouTube URL, local video file (%s), or local audio file (%s)", strings.Join(acceptedLocalVideoExtensions, ", "), strings.Join(acceptedLocalAudioExtensions, ", "))
	}
	info, err := os.Stat(input)
	if err != nil {
		return mediaSource{}, fmt.Errorf("%s is not readable: %w", localSourceKind(input), err)
	}
	if info.IsDir() {
		return mediaSource{}, fmt.Errorf("%s must be a file, not a directory", localSourceKind(input))
	}
	if isAcceptedLocalAudioExtension(input) {
		return mediaSource{kind: mediaSourceLocalAudio, value: input}, nil
	}
	return mediaSource{kind: mediaSourceLocalVideo, value: input}, nil
}

var acceptedLocalVideoExtensions = []string{".mp4", ".mov", ".mkv", ".webm", ".avi", ".m4v"}
var acceptedLocalAudioExtensions = []string{".mp3", ".wav", ".flac", ".ogg"}

func isAcceptedLocalVideoExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, accepted := range acceptedLocalVideoExtensions {
		if ext == accepted {
			return true
		}
	}
	return false
}

func isAcceptedLocalAudioExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, accepted := range acceptedLocalAudioExtensions {
		if ext == accepted {
			return true
		}
	}
	return false
}

func localSourceKind(path string) string {
	if isAcceptedLocalAudioExtension(path) {
		return "local audio source"
	}
	return "local video source"
}

func isSubtitleSourcePath(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".srt")
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

var acceptedTargetLanguages = map[string]struct{}{
	"ar": {}, "bn": {}, "br": {}, "ca": {}, "cs": {}, "da": {}, "de": {}, "el": {}, "en": {}, "es": {},
	"fa": {}, "fi": {}, "fr": {}, "gu": {}, "he": {}, "hi": {}, "hr": {}, "id": {}, "it": {}, "ja": {},
	"kn": {}, "ko": {}, "lo": {}, "mr": {}, "ms": {}, "ne": {}, "nl": {}, "no": {}, "pl": {}, "pt": {},
	"pa": {}, "ro": {}, "ru": {}, "sr": {}, "sv": {}, "ta": {}, "te": {}, "th": {}, "tl": {}, "tr": {},
	"uk": {}, "ur": {}, "vi": {}, "zh": {},
}

func isAcceptedTargetLanguage(language string) bool {
	_, ok := acceptedTargetLanguages[language]
	return ok
}
