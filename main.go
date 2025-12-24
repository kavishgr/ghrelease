package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/kavishgr/ghrelease/github"
	"github.com/kavishgr/ghrelease/options"
	"github.com/kavishgr/ghrelease/utils"
)

func main() {

	var (
		opts           = options.ParseFlags()
		skipextraction = opts.SkipExtraction
		token          = opts.GHToken
		tempdir        = opts.TempDir
		ost, arch      = utils.OsInfo()
		regex          = utils.SetRegex(ost, arch)
		stdInUrls      = make(chan string)
		jobs           sync.WaitGroup
		version        = "0.1.3"
	)

	if opts.Version {
		fmt.Println("ghrelease version: ", version)
		os.Exit(0)
	}

	if token == "" {
		fmt.Println("GITHUB_TOKEN environment variable is not found.")
		fmt.Println("Nor is -ghtoken provided on the command line.")
		fmt.Println("")
		fmt.Println("Run 'ghrelease -h'")
		fmt.Println("Or browse to: 'https://github.com/kavishgr/ghrelease'")
		os.Exit(1)
	}

	if err := github.ValidateToken(token); err != nil {
		fmt.Fprintf(os.Stderr, "Github token validation failed, %v\n", err)
		fmt.Fprintln(os.Stderr, "Please check your toekn and try again.")
		os.Exit(1)
	}

	if len(os.Args) == 1 {
		fmt.Println("No arguments were provided.")
		fmt.Println("Run: 'ghrelease -h'")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nCancelling operations...")
		cancel()
	}()

	go utils.ScanStdIn(stdInUrls)

	if opts.List {
		for c := 0; c < opts.Concurrency; c++ {
			jobs.Add(1)
			go github.FetchGithubReleaseUrl(ctx, stdInUrls, &jobs, regex, token)
		}
	}

	if opts.Download {
		if err := os.MkdirAll(tempdir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", tempdir, err)
			os.Exit(1)
		}

		for c := 0; c < opts.Concurrency; c++ {
			jobs.Add(1)
			go github.DownloadRelease(ctx, stdInUrls, &jobs, token, tempdir, skipextraction)
		}
	}

	jobs.Wait() // wait for above jobs to finish

	// check if operation was cancelled
	if ctx.Err() != nil {
		fmt.Println("Operation cancelled. Partial files removed.")
		os.Exit(130) // exit code for SIGINT
	}

	switch {

	case opts.List:
		return

	case skipextraction:
		fmt.Println("Archives saved in: ", tempdir)

	default:
		if err := cleanup(tempdir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cleanup failed in %s: %v\n", opts.TempDir, err)
		}
		fmt.Println("")
		fmt.Println("Binaries extracted to: ", tempdir)
	}
}
