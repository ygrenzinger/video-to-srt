package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"

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

type cliConfig struct {
	Transcribe transcribeCommand `cmd:"" default:"withargs" help:"Turn a Media Source into SRT subtitles."`
	Translate  translateCommand  `cmd:"" help:"Translate an existing Subtitle Source."`
	Version    bool              `help:"Print version and exit."`
}

type transcribeCommand struct {
	MediaSource               string `arg:"" name:"media-source" help:"YouTube Source, Local Video Source, or Local Audio Source."`
	OutputDir                 string `name:"output-dir" default:"." type:"path" help:"Directory for generated files."`
	Provider                  string `name:"provider" enum:"voxtral,grok" default:"voxtral" help:"Transcription Provider."`
	Model                     string `name:"model" help:"Transcription Provider model id."`
	TargetLanguage            string `name:"target-language" help:"Translate Subtitle Cues to a supported Target Language code."`
	TranslationProvider       string `name:"translation-provider" help:"Translation Provider."`
	TranslationModel          string `name:"translation-model" help:"Translation Provider model id."`
	YouTubeCookies            string `name:"youtube-cookies" help:"Cookies file to pass to yt-dlp."`
	YouTubeCookiesFromBrowser string `name:"youtube-cookies-from-browser" help:"Browser cookie store to pass to yt-dlp, such as chrome or firefox."`
	Quiet                     bool   `name:"quiet" help:"Print only generated SRT paths to stdout."`
}

type translateCommand struct {
	SubtitleSource      string `arg:"" name:"subtitle-source" type:"path" help:"Existing SRT file to translate."`
	TargetLanguage      string `name:"target-language" required:"" help:"Translate Subtitle Cues to a supported Target Language code."`
	TranslationProvider string `name:"translation-provider" help:"Translation Provider."`
	TranslationModel    string `name:"translation-model" help:"Translation Provider model id."`
	Quiet               bool   `name:"quiet" help:"Print only generated SRT paths to stdout."`
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

	if wantsHelp(argv) {
		printHelp(stdout)
		return 0
	}
	if wantsVersion(argv) {
		fmt.Fprintln(stdout, Version)
		return 0
	}

	cli, selectedCommand, err := parseCLI(argv, stdout, stderr)
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 2
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
	if selectedCommand == "translate" {
		return runSubtitleSource(ctx, cli.Translate.SubtitleSource, cli.Translate.TargetLanguage, cli.Translate.TranslationProvider, cli.Translate.TranslationModel, cli.Translate.Quiet, stdout, stderr, translate)
	}
	cmd := cli.Transcribe
	if cmd.TargetLanguage != "" && !isAcceptedTargetLanguage(cmd.TargetLanguage) {
		fmt.Fprintf(stderr, "Error: unsupported target language %q\n", cmd.TargetLanguage)
		return 1
	}
	if cmd.TranslationProvider != "" && cmd.TranslationProvider != "mistral" && cmd.TranslationProvider != "grok" {
		fmt.Fprintf(stderr, "Error: unsupported translation provider %q\n", cmd.TranslationProvider)
		return 1
	}
	mediaSource, err := classifyMediaSource(cmd.MediaSource)
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if mediaSource.kind != mediaSourceYouTube && (cmd.YouTubeCookies != "" || cmd.YouTubeCookiesFromBrowser != "") {
		fmt.Fprintln(stderr, "Error: YouTube cookie options can only be used with YouTube Sources")
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
	if !cmd.Quiet {
		fmt.Fprintln(stderr, "Preparing Media Source...")
	}
	var audioPath string
	cleanupAudio := true
	switch mediaSource.kind {
	case mediaSourceYouTube:
		audioPath, err = downloadAudio(ctx, SourceRequest{URL: mediaSource.value, OutputDir: cmd.OutputDir, Cookies: cmd.YouTubeCookies, CookiesFromBrowser: cmd.YouTubeCookiesFromBrowser})
	case mediaSourceLocalVideo:
		audioPath, err = extractLocalAudio(ctx, LocalVideoRequest{Path: mediaSource.value, OutputDir: cmd.OutputDir})
	case mediaSourceLocalAudio:
		audioPath = mediaSource.value
		cleanupAudio = false
	}
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if !cmd.Quiet {
		fmt.Fprintf(stderr, "Transcribing with %s...\n", providerDisplayName(cmd.Provider))
	}
	outputPath := srtPath(audioPath, cmd.Provider)
	if mediaSource.kind == mediaSourceLocalAudio {
		outputPath = outputSRTPath(cmd.OutputDir, audioPath, cmd.Provider)
	}
	cues, transcribeErr := transcribe(ctx, TranscriptionRequest{Provider: cmd.Provider, Model: cmd.Model, AudioPath: audioPath, OutputPath: outputPath})
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
	if cmd.TargetLanguage != "" {
		selectedTranslationProvider := cmd.TranslationProvider
		if selectedTranslationProvider == "" {
			selectedTranslationProvider = defaultTranslationProvider(cmd.Provider)
		}
		translatedPath := translatedSRTPath(outputPath, cmd.TargetLanguage)
		translatedCues, err := translate(ctx, TranslationRequest{Provider: selectedTranslationProvider, Model: cmd.TranslationModel, TargetLanguage: cmd.TargetLanguage, Cues: cues, OutputPath: translatedPath})
		if err != nil {
			fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		if err := subtitles.AtomicWriteSRT(translatedPath, translatedCues); err != nil {
			fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		if cmd.Quiet {
			fmt.Fprintln(stdout, outputPath)
			fmt.Fprintln(stdout, translatedPath)
		} else {
			fmt.Fprintln(stderr, "Output:", outputPath)
			fmt.Fprintln(stderr, "Translated output:", translatedPath)
		}
		return 0
	}
	if cmd.Quiet {
		fmt.Fprintln(stdout, outputPath)
	} else {
		fmt.Fprintln(stderr, "Output:", outputPath)
	}
	return 0
}

func parseCLI(argv []string, stdout, stderr io.Writer) (cliConfig, string, error) {
	var cli cliConfig
	parser, err := kong.New(&cli,
		kong.Name("video-to-srt"),
		kong.Description("Turn a Media Source into SRT subtitles, or translate an existing Subtitle Source."),
		kong.Writers(stdout, stderr),
		kong.Exit(func(int) {}),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:        true,
			FlagsLast:      true,
			WrapUpperBound: 100,
		}),
	)
	if err != nil {
		return cliConfig{}, "", err
	}
	parsed, err := parser.Parse(argv)
	if err != nil {
		return cliConfig{}, "", err
	}
	command := parsed.Command()
	if strings.HasPrefix(command, "translate") {
		return cli, "translate", nil
	}
	return cli, "transcribe", nil
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

func wantsHelp(argv []string) bool {
	for _, arg := range argv {
		if arg == "-h" || arg == "-help" || arg == "--help" {
			return true
		}
	}
	return false
}

func wantsVersion(argv []string) bool {
	for _, arg := range argv {
		if arg == "--version" {
			return true
		}
	}
	return false
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  video-to-srt [transcribe] [options] <media-source>
  video-to-srt translate [options] <subtitle-source.srt>

Turn a Media Source into SRT subtitles, or translate an existing Subtitle Source.

Accepted sources:
  YouTube Source:      https://youtube.com/..., https://www.youtube.com/..., https://m.youtube.com/..., or https://youtu.be/...
  Local Video Source:  readable local file with extension .mp4, .mov, .mkv, .webm, .avi, or .m4v
  Local Audio Source:  readable local file with extension .mp3, .wav, .flac, or .ogg
  Subtitle Source:     readable local .srt file; requires --target-language

Examples:
  video-to-srt 'https://www.youtube.com/watch?v=abc123'
  video-to-srt transcribe ./talk.final.mp4
  video-to-srt ./talk.final.mp4
  video-to-srt --provider grok ./talk.final.mp3
  video-to-srt --target-language fr ./talk.final.mp4
  video-to-srt translate --target-language fr ./talk.final.voxtral.srt

Requirements:
  YouTube Sources require yt-dlp on PATH.
  Local Video Sources require ffmpeg on PATH.
  Local Audio Sources do not require yt-dlp or ffmpeg.
  Voxtral and Mistral translation use MISTRAL_API_KEY.
  Grok transcription and translation use XAI_API_KEY.

Options:
  --output-dir <dir>                  Directory for generated files. Defaults to the current directory.
  --provider <voxtral|grok>           Transcription Provider. Defaults to voxtral.
  --model <model-id>                  Transcription Provider model id. Defaults to the provider default.
  --target-language <code>            Translate Subtitle Cues to a supported Target Language code.
  --translation-provider <mistral|grok>
                                      Translation Provider. Defaults to mistral for voxtral and grok for grok.
  --translation-model <model-id>      Translation Provider model id. Defaults to the provider default.
  --youtube-cookies <path>            Cookies file to pass to yt-dlp. Valid only for YouTube Sources.
  --youtube-cookies-from-browser <id> Browser cookie store to pass to yt-dlp, such as chrome or firefox.
  --quiet                             Print only generated SRT paths to stdout.
  --version                           Print version and exit.
  --help                              Print this help and exit.

Commands:
  transcribe                           Turn a Media Source into SRT subtitles. Optional by default.
  translate                            Translate an existing Subtitle Source. Required for .srt inputs.
`)
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
	if !isAcceptedTargetLanguage(targetLanguage) {
		fmt.Fprintf(stderr, "Error: unsupported target language %q\n", targetLanguage)
		return 1
	}
	if !isSubtitleSourcePath(path) {
		fmt.Fprintln(stderr, "Error: Subtitle Source must be a .srt file")
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
