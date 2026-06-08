package source

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type LocalRequest struct {
	Path      string
	OutputDir string
}

func ExtractLocalAudio(ctx context.Context, req LocalRequest, runner Runner) (string, error) {
	if runner == nil {
		runner = ExecRunner{}
	}
	if _, err := runner.LookPath("ffmpeg"); err != nil {
		return "", errors.New("ffmpeg is required for local video files")
	}
	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = "."
	}
	outputPath := filepath.Join(outputDir, strings.TrimSuffix(filepath.Base(req.Path), filepath.Ext(req.Path))+".mp3")
	args := []string{
		"-y",
		"-i", req.Path,
		"-vn",
		"-codec:a", "libmp3lame",
		"-q:a", "2",
		outputPath,
	}
	result, err := runner.Run(ctx, "ffmpeg", args)
	if err != nil {
		detail := strings.TrimSpace(result.Stderr)
		if detail == "" {
			detail = strings.TrimSpace(result.Stdout)
		}
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("local video audio extraction failed: %s", detail)
	}
	return outputPath, nil
}
