package main

import (
	"context"
	"os"

	"video-to-srt/internal/app"
)

var version = "dev"

func main() {
	app.Version = version
	os.Exit(app.Run(context.Background(), os.Args[1:], app.Streams{}, app.Runner{}))
}
