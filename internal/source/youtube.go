package source

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Request struct {
	URL                string
	OutputDir          string
	Cookies            string
	CookiesFromBrowser string
}

type RunResult struct {
	Stdout string
	Stderr string
}

type Runner interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, name string, args []string) (RunResult, error)
}

type ExecRunner struct{}

func (ExecRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (ExecRunner) Run(ctx context.Context, name string, args []string) (RunResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return RunResult{Stdout: stdout.String(), Stderr: stderr.String()}, err
}

func DownloadAudio(ctx context.Context, req Request, runner Runner) (string, error) {
	if runner == nil {
		runner = ExecRunner{}
	}
	if _, err := runner.LookPath("yt-dlp"); err != nil {
		return "", errors.New("yt-dlp is required for YouTube URLs")
	}
	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = "."
	}
	args := []string{
		"--no-playlist",
		"--extract-audio",
		"--audio-format", "mp3",
		"--paths", outputDir,
	}
	if req.Cookies != "" {
		args = append(args, "--cookies", req.Cookies)
	}
	if req.CookiesFromBrowser != "" {
		args = append(args, "--cookies-from-browser", req.CookiesFromBrowser)
	}
	args = append(args,
		"--print", "after_move:filepath",
		"-o", "%(title).200B [%(id)s].%(ext)s",
		req.URL,
	)
	result, err := runner.Run(ctx, "yt-dlp", args)
	if err != nil {
		detail := strings.TrimSpace(result.Stderr)
		if detail == "" {
			detail = strings.TrimSpace(result.Stdout)
		}
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("YouTube download failed: %s", detail)
	}
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if path := strings.TrimSpace(lines[i]); path != "" {
			return path, nil
		}
	}
	return "", errors.New("YouTube download failed: yt-dlp did not report an output file")
}
