package main

import (
	"fmt"
	"os"
	"sync"

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

	if len(os.Args) == 1 {
		fmt.Println("No arguments were provided.")
		fmt.Println("Run: 'ghrelease -h'")
		os.Exit(1)
	}

	go utils.ScanStdIn(stdInUrls)

	if opts.List {
		for c := 0; c < opts.Concurrency; c++ {
			jobs.Add(1)
			go github.FetchGithubReleaseUrl(stdInUrls, &jobs, regex, token)
		}
	}

	if opts.Download {
		if err := os.MkdirAll(tempdir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", tempdir, err)
			os.Exit(1)
		}

		for c := 0; c < opts.Concurrency; c++ {
			jobs.Add(1)
			go github.DownloadRelease(stdInUrls, &jobs, token, tempdir, skipextraction)
		}
	}

	jobs.Wait() // wait for above jobs to finish

	switch {

	case opts.List:
		return

	case skipextraction:
		fmt.Println("Archives are inside: ", tempdir)

	default:
		cleanup(tempdir)
		fmt.Println("")
		fmt.Println("All Binaries are inside: ", tempdir)
	}
}
