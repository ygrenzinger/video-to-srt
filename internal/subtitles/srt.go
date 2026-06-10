package subtitles

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

func ParseSRT(input string) ([]Cue, error) {
	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	blocks := strings.Split(strings.TrimSpace(normalized), "\n\n")
	cues := []Cue{}
	for i, block := range blocks {
		lines := strings.Split(block, "\n")
		if len(lines) < 3 {
			return nil, fmt.Errorf("cue %d is incomplete", i+1)
		}
		if _, err := strconv.Atoi(strings.TrimSpace(lines[0])); err != nil {
			return nil, fmt.Errorf("cue %d index is invalid", i+1)
		}
		timing := strings.Split(lines[1], " --> ")
		if len(timing) != 2 {
			return nil, fmt.Errorf("cue %d timing separator is invalid", i+1)
		}
		start, err := parseTimestamp(strings.TrimSpace(timing[0]))
		if err != nil {
			return nil, fmt.Errorf("cue %d start timestamp is invalid: %w", i+1, err)
		}
		end, err := parseTimestamp(strings.TrimSpace(timing[1]))
		if err != nil {
			return nil, fmt.Errorf("cue %d end timestamp is invalid: %w", i+1, err)
		}
		text := strings.TrimSpace(strings.Join(lines[2:], "\n"))
		cue := Cue{StartMS: start, EndMS: end, Text: text}
		if _, err := FormatSRT([]Cue{cue}); err != nil {
			return nil, err
		}
		cues = append(cues, cue)
	}
	return cues, nil
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

func parseTimestamp(value string) (int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("expected HH:MM:SS,mmm")
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	secondParts := strings.Split(parts[2], ",")
	if len(secondParts) != 2 {
		return 0, fmt.Errorf("expected seconds and milliseconds")
	}
	seconds, err := strconv.Atoi(secondParts[0])
	if err != nil {
		return 0, err
	}
	millis, err := strconv.Atoi(secondParts[1])
	if err != nil {
		return 0, err
	}
	if hours < 0 || minutes < 0 || minutes > 59 || seconds < 0 || seconds > 59 || millis < 0 || millis > 999 {
		return 0, fmt.Errorf("timestamp component out of range")
	}
	return ((hours*60+minutes)*60+seconds)*1000 + millis, nil
}
