package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"lingq_upload/internal/downloader"
)

func main() {
	var (
		bookInput  string
		outputRoot string
		skipUnzip  bool
	)

	flag.StringVar(&bookInput, "book", "", "Book slug or full URL (e.g. /book/body-on-the-rocks-denise-kirby)")
	flag.StringVar(&bookInput, "b", "", "Book slug or full URL (shorthand)")
	flag.StringVar(&outputRoot, "out", ".", "Directory where the downloaded book will be stored")
	flag.BoolVar(&skipUnzip, "skip-unzip", false, "Skip extracting the mp3 zip archive")
	flag.Parse()

	if bookInput == "" {
		fmt.Fprintln(os.Stderr, "missing required -book argument")
		flag.Usage()
		os.Exit(1)
	}

	manager := downloader.NewManager(outputRoot)
	manager.RegisterProvider(downloader.NewEnglishEReaderProvider(downloader.EnglishEReaderOptions{
		SkipUnzip: skipUnzip,
	}))

	ctx := context.Background()
	result, err := manager.Download(ctx, bookInput)
	if err != nil {
		log.Fatalf("download failed: %v", err)
	}

	absDir, _ := filepath.Abs(result.OutputDir)
	log.Printf("Downloaded by provider %q into %s", result.Provider, absDir)
	for _, file := range result.Files {
		log.Printf(" - %s", file)
	}
	if result.MetadataPath != "" {
		log.Printf("Metadata written to %s", result.MetadataPath)
	}
}
