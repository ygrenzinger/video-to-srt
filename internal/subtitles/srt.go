package subtitles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Cue struct {
	StartMS int
	EndMS   int
	Text    string
}

func FormatSRT(cues []Cue) (string, error) {
	blocks := make([]string, 0, len(cues))
	for i, cue := range cues {
		if cue.StartMS < 0 {
			return "", fmt.Errorf("cue %d start cannot be negative", i+1)
		}
		if cue.EndMS <= cue.StartMS {
			return "", fmt.Errorf("cue %d end must be after start", i+1)
		}
		text := strings.TrimSpace(cue.Text)
		if text == "" {
			return "", fmt.Errorf("cue %d text cannot be empty", i+1)
		}
		start := formatTimestamp(cue.StartMS)
		end := formatTimestamp(cue.EndMS)
		blocks = append(blocks, fmt.Sprintf("%d\n%s --> %s\n%s", i+1, start, end, text))
	}
	if len(blocks) == 0 {
		return "", nil
	}
	return strings.Join(blocks, "\n\n") + "\n", nil
}

func AtomicWriteSRT(path string, cues []Cue) error {
	out, err := FormatSRT(cues)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(out); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func formatTimestamp(ms int) string {
	totalSeconds := ms / 1000
	millis := ms % 1000
	seconds := totalSeconds % 60
	totalMinutes := totalSeconds / 60
	minutes := totalMinutes % 60
	hours := totalMinutes / 60
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, millis)
}
