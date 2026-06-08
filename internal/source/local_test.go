package source

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExtractLocalAudioUsesFFmpegMP3Extraction(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "talk.final.mp4")
	audioPath := filepath.Join(dir, "talk.final.mp3")
	runner := &fakeRunner{}

	got, err := ExtractLocalAudio(context.Background(), LocalRequest{
		Path:      videoPath,
		OutputDir: dir,
	}, runner)

	if err != nil {
		t.Fatalf("ExtractLocalAudio() err = %v", err)
	}
	if got != audioPath {
		t.Fatalf("ExtractLocalAudio() = %q, want %q", got, audioPath)
	}
	if runner.name != "ffmpeg" {
		t.Fatalf("runner name = %q", runner.name)
	}
	wantArgs := []string{
		"-y",
		"-i", videoPath,
		"-vn",
		"-codec:a", "libmp3lame",
		"-q:a", "2",
		audioPath,
	}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args = %#v\nwant %#v", runner.args, wantArgs)
	}
}

func TestExtractLocalAudioReportsMissingAndFailingFFmpeg(t *testing.T) {
	_, err := ExtractLocalAudio(context.Background(), LocalRequest{Path: "clip.mp4"}, &fakeRunner{pathErr: errors.New("missing")})
	if err == nil || !strings.Contains(err.Error(), "ffmpeg is required") {
		t.Fatalf("missing ffmpeg err = %v", err)
	}

	_, err = ExtractLocalAudio(context.Background(), LocalRequest{Path: "clip.mp4"}, &fakeRunner{runErr: errors.New("exit 1"), stderr: "invalid data"})
	if err == nil || !strings.Contains(err.Error(), "invalid data") {
		t.Fatalf("failing ffmpeg err = %v", err)
	}
}
