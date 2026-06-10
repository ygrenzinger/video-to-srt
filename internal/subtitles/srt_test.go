package subtitles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatSRTFormatsSubtitleCues(t *testing.T) {
	got, err := FormatSRT([]Cue{
		{StartMS: 1250, EndMS: 3500, Text: "Hello"},
		{StartMS: 62000, EndMS: 65005, Text: "World"},
	})

	if err != nil {
		t.Fatalf("FormatSRT() err = %v", err)
	}
	want := "1\n00:00:01,250 --> 00:00:03,500\nHello\n\n2\n00:01:02,000 --> 00:01:05,005\nWorld\n"
	if got != want {
		t.Fatalf("FormatSRT() = %q\nwant %q", got, want)
	}
}

func TestFormatSRTRejectsInvalidTiming(t *testing.T) {
	_, err := FormatSRT([]Cue{{StartMS: 1000, EndMS: 1000, Text: "bad"}})
	if err == nil || !strings.Contains(err.Error(), "end must be after start") {
		t.Fatalf("FormatSRT() err = %v", err)
	}
}

func TestAtomicWriteSRTDoesNotOverwriteExistingFileOnInvalidCue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.srt")
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := AtomicWriteSRT(path, []Cue{{StartMS: 1000, EndMS: 1000, Text: "bad"}})

	if err == nil {
		t.Fatal("expected error")
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "existing\n" {
		t.Fatalf("file = %q, want existing contents", string(data))
	}
}

func TestParseSRTParsesSubtitleCues(t *testing.T) {
	input := "1\n00:00:01,000 --> 00:00:02,500\nHello\nworld\n\n2\n00:00:03,000 --> 00:00:04,000\nGoodbye\n"

	got, err := ParseSRT(input)

	if err != nil {
		t.Fatalf("ParseSRT() err = %v", err)
	}
	want := []Cue{
		{StartMS: 1000, EndMS: 2500, Text: "Hello\nworld"},
		{StartMS: 3000, EndMS: 4000, Text: "Goodbye"},
	}
	if len(got) != len(want) {
		t.Fatalf("ParseSRT() len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("cue %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestParseSRTRejectsInvalidTiming(t *testing.T) {
	_, err := ParseSRT("1\n00:00:02,000 --> 00:00:01,000\nbad\n")

	if err == nil {
		t.Fatal("ParseSRT() err = nil")
	}
}
