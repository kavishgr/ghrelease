package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/kavishgr/ghrelease/github"
	"github.com/kavishgr/ghrelease/options"
	"github.com/kavishgr/ghrelease/utils"
	"github.com/vbauerster/mpb/v8"
)

func printSummary(elapsed time.Duration, tempdir string) {
	downloaded, failed := github.GetDownloadStats()
	elapsed = elapsed.Round(time.Millisecond)

	fmt.Printf("\n%-12s %s\n", "Finished in:", elapsed)

	summary := fmt.Sprintf("%d downloaded, %d failed", downloaded, failed)
	errorLogPath := filepath.Join(tempdir, "error.log")

	// If there are failures, append the log location
	if failed > 0 {
		summary = fmt.Sprintf("%s (see %s)", summary, errorLogPath)
	} else {
		// remove empty log file
		// _ = means this might return an error if file don't exist
		// it's not an issue
		_ = os.Remove(errorLogPath)
	}

	fmt.Printf("%-12s %s\n", "Summary:", summary)
}

// Version information (set by goreleaser)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {

	var (
		opts           = options.ParseFlags()
		skipextraction = opts.SkipExtraction
		token          = os.Getenv("GITHUB_TOKEN")
		tempdir        = opts.TempDir
		stdInUrls      = make(chan string)
		jobs           sync.WaitGroup
	)

	if opts.Version {
		fmt.Printf("ghrelease version %s (commit %s, built at %s)\n", version, commit, date)
		os.Exit(0)
	}

	if token == "" {
		fmt.Println("GITHUB_TOKEN environment variable is not found.")
		fmt.Println("Run 'ghrelease -h'")
		fmt.Println("Or browse to: 'https://github.com/kavishgr/ghrelease'")
		os.Exit(1)
	}

	client, err := github.NewClient(token, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client %s\n", err)
		os.Exit(1)
	}

	if err := client.ValidateToken(); err != nil {
		fmt.Fprintf(os.Stderr, "Github token validation failed, %v\n", err)
		fmt.Fprintln(os.Stderr, "Please check your token and try again.")
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
			go client.FetchGithubReleaseURLs(ctx, stdInUrls, &jobs)
		}
		jobs.Wait()
	}

	if opts.Download {
		if err := os.MkdirAll(tempdir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", tempdir, err)
			os.Exit(1)
		}

		logFile, err := github.InitErrorLog(tempdir)
		if err != nil {
			fmt.Printf("Warning: Could not create log in %s: %v\n", tempdir, err)
		} else {
			defer logFile.Close()
		}

		start := time.Now()

		p := mpb.NewWithContext(
			ctx,
			mpb.WithRefreshRate(180*time.Millisecond),
			mpb.WithOutput(os.Stderr),
		)

		for c := 0; c < opts.Concurrency; c++ {
			jobs.Add(1)
			go client.DownloadReleases(p, ctx, stdInUrls, &jobs, tempdir, skipextraction, logFile)
		}
		jobs.Wait()
		p.Wait()

		duration := time.Since(start)

		// check if operation was cancelled
		if ctx.Err() != nil {
			fmt.Println("Operation cancelled. All files removed.")
			os.RemoveAll(tempdir)
			os.Exit(130) // exit code for SIGINT
		}

		fmt.Println()
		printSummary(duration, tempdir)

		if skipextraction {
			fmt.Println("Archives saved in: ", tempdir)
		} else {
			if err := utils.Cleanup(tempdir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cleanup failed in %s: %v\n", opts.TempDir, err)
			}
			fmt.Println("Binaries extracted to: ", tempdir)
		}
	}
}
