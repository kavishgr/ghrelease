package github

import (
	"context"
	// "fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"

	// "github.com/k0kubun/go-ansi"
	"github.com/kavishgr/ghrelease/utils"
	"github.com/schollz/progressbar/v3"
)

// DownloadReleases downloads release assets from URLs received via urlsChan.
func (c *Client) DownloadReleases(ctx context.Context, urlsChan chan string, job *sync.WaitGroup, tempdir string, skipextraction bool) {
	defer job.Done()

	downloadAndProcessFile := func(u string) {
		file := path.Base(u)
		src := filepath.Join(tempdir, file)

		req := c.craftRequest(u)
		req = req.WithContext(ctx)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("Failed to download %s: %v", file, err)
			}
			return
		}
		defer resp.Body.Close()

		f, err := os.OpenFile(src, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Failed to create file %s: %v", file, err)
			return
		}
		defer f.Close()
		bar := progressbar.DefaultBytes(
			resp.ContentLength,
			file,
		)

		// bar := progressbar.NewOptions64(resp.ContentLength,
		// 	progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		// 	progressbar.OptionEnableColorCodes(true),
		// 	progressbar.OptionClearOnFinish(),
		// 	progressbar.OptionSetElapsedTime(true),
		// 	// progressbar.OptionShowElapsedTimeOnFinish(),
		// 	// progressbar.OptionSetPredictTime(false),
		// 	progressbar.OptionShowBytes(true),
		// 	progressbar.OptionSetWidth(15),
		// 	progressbar.OptionSetDescription(fmt.Sprintf("%s", file)),
		// 	progressbar.OptionSetTheme(progressbar.Theme{
		// 		Saucer:        "[green]=[reset]",
		// 		SaucerHead:    "[green]>[reset]",
		// 		SaucerPadding: " ",
		// 		BarStart:      "[",
		// 		BarEnd:        "]",
		// 	}))
		// look like this:
		// fx_darwin_arm64  15% [=>   ] (148 kB/s) [14s:1m41s]

		io.Copy(io.MultiWriter(f, bar), resp.Body)
		// bar.Clear()
		// bar.Reset()
		bar.Finish()

		if skipextraction {
			// fmt.Printf("Downloaded: %s\n", file)
			bar.Close()
			return
		}

		if err := utils.Extractor(src, tempdir); err != nil {
			log.Printf("Extraction failed for %s: %v", file, err)
			return
		}

		bar.Close()
		// fmt.Printf("Downloaded and Extracted: %s\n", file)
	}

	// iterate over urls sent by stdin
	for u := range urlsChan {
		select {
		case <-ctx.Done():
			return
		default:
			downloadAndProcessFile(u)
		}
	}
}
