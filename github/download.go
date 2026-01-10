package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	// "log"
	"os"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/kavishgr/ghrelease/utils"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var (
	downloadedCount atomic.Int32
	failedCount     atomic.Int32
	logMu           sync.Mutex
)

const (
	maxNameWidth int = 30
)

func InitErrorLog(dir string) (*os.File, error) {
	logPath := filepath.Join(dir, "error.log")
	return os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

func GetDownloadStats() (download, failed int32) {
	return downloadedCount.Load(), failedCount.Load()
}

func (c *Client) DownloadReleases(p *mpb.Progress, ctx context.Context, urlsChan chan string, job *sync.WaitGroup, tempdir string, skipextraction bool, logFile *os.File) {
	defer job.Done()

	downloadedCount.Store(0)
	failedCount.Store(0)

	downloadAndProcessFile := func(u string) {
		file := path.Base(u)
		src := filepath.Join(tempdir, file)

		handleError := func(err error, stage string) {
			failedCount.Add(1)
			fmt.Fprintf(p, "✗ %-s (%s failed)\n", file, stage)
			if logFile != nil {
				logMu.Lock()
				defer logMu.Unlock()
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				fmt.Fprintf(logFile, "[%s] %s ERROR: %s -> %v\n", timestamp, stage, file, err)
			}
		}

		req := c.craftRequest(u)
		req = req.WithContext(ctx)
		resp, err := c.httpClient.Do(req)

		if err != nil {
			if ctx.Err() == nil {
				handleError(err, "Download")
			}
			return
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			handleError(fmt.Errorf("server returned %s", resp.Status), "Download")
			return
		}

		f, err := os.OpenFile(src, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			handleError(err, "File creation")
			return
		}
		defer f.Close()

		displayFile := file
		if len(file) > maxNameWidth {
			displayFile = file[:maxNameWidth-3] + "..."
		}

		total := resp.ContentLength
		queue := make([]*mpb.Bar, 2)
		// download bar
		queue[0] = p.AddBar(total,
			mpb.PrependDecorators(
				decor.Name(displayFile, decor.WC{W: maxNameWidth, C: 1}),
				decor.Name("downloading", decor.WCSyncSpaceR),
				decor.Counters(decor.SizeB1024(0), "% .2f / % .2f", decor.WCSyncWidth),
			),
			mpb.AppendDecorators(
				decor.EwmaSpeed(decor.SizeB1024(0), "% .2f", 60, decor.WCSyncWidth),
			),
			mpb.BarRemoveOnComplete(), // HIDE when done
		)
		// extraction bar
		queue[1] = p.AddBar(1,
			mpb.BarQueueAfter(queue[0]),
			mpb.BarFillerClearOnComplete(),
			mpb.PrependDecorators(
				decor.Name(displayFile, decor.WC{W: maxNameWidth, C: 1}),
				decor.Name("extracting", decor.WCSyncSpaceR),
			),
			mpb.BarRemoveOnComplete(), // HIDE when done
		)

		proxyReader := queue[0].ProxyReader(resp.Body)
		_, err = io.Copy(f, proxyReader)
		proxyReader.Close()

		if err != nil {
			queue[0].Abort(true)
			queue[1].Abort(true)
			handleError(err, "Network")
			return
		}

		if !skipextraction {
			if err := utils.Extractor(src, tempdir); err != nil {
				queue[1].Abort(true)
				handleError(err, "Extraction")
				return
			}
		}

		// remove bar and print filename
		queue[1].Increment()

		fmt.Fprintf(p, "✓ %-s\n", file)

		downloadedCount.Add(1)
	}

	for u := range urlsChan {
		select {
		case <-ctx.Done():
			return
		default:
			downloadAndProcessFile(u)
		}
	}
}
