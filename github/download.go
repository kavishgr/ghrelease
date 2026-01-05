package github

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/kavishgr/ghrelease/utils"
	// "github.com/schollz/progressbar/v3"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var (
	downloadedCount atomic.Int32
	failedCount     atomic.Int32
)

const (
	maxNameWidth int = 30
	counterWidth int = 21 // Enough space for "xxx.xx MiB / xxx.xx MiB"
	speedWidth   int = 12
)

func GetDownloadStats() (download, failed int32) {
	return downloadedCount.Load(), failedCount.Load()
}

// DownloadReleases downloads release assets from URLs received via urlsChan.
func (c *Client) DownloadReleases(p *mpb.Progress, ctx context.Context, urlsChan chan string, job *sync.WaitGroup, tempdir string, skipextraction bool) {
	defer job.Done()

	downloadedCount.Store(0)
	failedCount.Store(0)

	downloadAndProcessFile := func(u string) {
		file := path.Base(u)
		src := filepath.Join(tempdir, file)

		req := c.craftRequest(u)
		req = req.WithContext(ctx)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("Failed to download %s: %v", file, err)
				failedCount.Add(1)
			}
			return
		}
		defer resp.Body.Close()

		f, err := os.OpenFile(src, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Failed to create file %s: %v", file, err)
			failedCount.Add(1)
			return
		}
		defer f.Close()

		displayFile := file
		if len(file) > maxNameWidth {
			displayFile = file[:maxNameWidth-3] + "..."
		}

		bar := p.AddBar(resp.ContentLength,
			mpb.PrependDecorators(
				// If DidentRight is undefined, use decor.W0 (zero width) or 0
				// This aligns the name to the left and prevents compiler errors
				decor.Name(displayFile+" ", decor.WC{W: maxNameWidth, C: 1}),

				// Using Counters with the unit type first solves the conversion error
				decor.Counters(decor.SizeB1024(0), "% .2f / % .2f", decor.WC{W: counterWidth, C: 1}),
			),
			mpb.AppendDecorators(
				// EwmaSpeed also needs the unit type first, then the format, then the age
				decor.EwmaSpeed(decor.SizeB1024(0), "% .2f", 60, decor.WC{W: speedWidth, C: 2}),
			),
		)
		proxyReader := bar.ProxyReader(resp.Body)
		defer proxyReader.Close()

		_, err = io.Copy(f, proxyReader)
		if err != nil {
			bar.Abort(true)
			fmt.Fprintf(p, "\x1b[31m✘ Network error on %s: %v\x1b[0m\n", file, err)
			failedCount.Add(1)
		}

		if skipextraction {
			downloadedCount.Add(1)
			return
		}

		if err := utils.Extractor(src, tempdir); err != nil {
			log.Printf("Extraction failed for %s: %v", file, err)
			failedCount.Add(1)
			return
		}

		downloadedCount.Add(1)
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
