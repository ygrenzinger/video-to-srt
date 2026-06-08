package source

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakeRunner struct {
	pathErr error
	runErr  error
	name    string
	args    []string
	stdout  string
	stderr  string
}

func (f *fakeRunner) LookPath(name string) (string, error) {
	if f.pathErr != nil {
		return "", f.pathErr
	}
	return "/usr/local/bin/" + name, nil
}

func (f *fakeRunner) Run(ctx context.Context, name string, args []string) (RunResult, error) {
	f.name = name
	f.args = append([]string(nil), args...)
	return RunResult{Stdout: f.stdout, Stderr: f.stderr}, f.runErr
}

func TestDownloadAudioUsesYTDLPDirectMP3Extraction(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "Example [abc123].mp3")
	runner := &fakeRunner{stdout: audioPath + "\n"}

	got, err := DownloadAudio(context.Background(), Request{
		URL:                "https://youtu.be/abc123",
		OutputDir:          dir,
		Cookies:            "cookies.txt",
		CookiesFromBrowser: "chrome",
	}, runner)

	if err != nil {
		t.Fatalf("DownloadAudio() err = %v", err)
	}
	if got != audioPath {
		t.Fatalf("DownloadAudio() = %q, want %q", got, audioPath)
	}
	if runner.name != "yt-dlp" {
		t.Fatalf("runner name = %q", runner.name)
	}
	wantArgs := []string{
		"--no-playlist",
		"--extract-audio",
		"--audio-format", "mp3",
		"--paths", dir,
		"--cookies", "cookies.txt",
		"--cookies-from-browser", "chrome",
		"--print", "after_move:filepath",
		"-o", "%(title).200B [%(id)s].%(ext)s",
		"https://youtu.be/abc123",
	}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args = %#v\nwant %#v", runner.args, wantArgs)
	}
}

func TestDownloadAudioReportsMissingAndFailingYTDLP(t *testing.T) {
	_, err := DownloadAudio(context.Background(), Request{URL: "https://youtu.be/abc123"}, &fakeRunner{pathErr: errors.New("missing")})
	if err == nil || !strings.Contains(err.Error(), "yt-dlp is required") {
		t.Fatalf("missing yt-dlp err = %v", err)
	}

	_, err = DownloadAudio(context.Background(), Request{URL: "https://youtu.be/abc123"}, &fakeRunner{runErr: errors.New("exit 1"), stderr: "blocked by YouTube"})
	if err == nil || !strings.Contains(err.Error(), "blocked by YouTube") {
		t.Fatalf("failing yt-dlp err = %v", err)
	}
}
