package main

import (
	"fmt"
	"os"

	downloader "github.com/zr/go-bunkatsu"
)

func main() {
	if err := downloader.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
