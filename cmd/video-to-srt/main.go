package main

import (
	"context"
	"os"

	"video-to-srt/internal/app"
)

func main() {
	os.Exit(app.Run(context.Background(), os.Args[1:], app.Streams{}, app.Runner{}))
}
