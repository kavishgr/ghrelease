package utils

import (
	"context"
	"fmt"
	"github.com/mholt/archives"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Extractor extracts archive files to tempdir by default.
// Supports multiple compression and archive formats including tar, zip, 7z, rar,
// and various compressed tar formats. The top-level directory is stripped from
// extracted files. Returns an error if extraction fails
// attempt is detected.
func Extractor(src, tempdir string) error {
	reader, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening file %s: %w", src, err)
	}
	defer reader.Close()

	ctx := context.Background()

	format, stream, err := archives.Identify(ctx, src, reader)
	if err != nil {
		// test: if strings.IndexByte(filepath.Base(src), '.') == -1
		return nil
	}

	ex, ok := format.(archives.Extractor)
	if !ok {
		// format identified but not extractable
		return nil
	}

	// Extract the archive
	err = ex.Extract(ctx, stream, func(ctx context.Context, f archives.FileInfo) error {
		pathInArchive := f.NameInArchive
		parts := strings.Split(filepath.ToSlash(pathInArchive), "/")

		var newPath string
		if len(parts) <= 1 {
			newPath = pathInArchive
		} else {
			newPath = filepath.Join(parts[1:]...)
		}

		if newPath == "" {
			return nil
		}

		newFilePath := filepath.Join(tempdir, filepath.Clean(newPath))

		cleanTempdir := filepath.Clean(tempdir)
		if !strings.HasPrefix(filepath.Clean(newFilePath), cleanTempdir) {
			return fmt.Errorf("invalid archive: path traversal attempt detected in %s", newPath)
		}

		if f.IsDir() {
			return os.MkdirAll(newFilePath, 0755)
		}

		if err := os.MkdirAll(filepath.Dir(newFilePath), 0755); err != nil {
			return fmt.Errorf("creating parent dirs for %s: %w", newPath, err)
		}

		content, err := f.Open()
		if err != nil {
			return fmt.Errorf("opening file %s from archive: %w", newPath, err)
		}
		defer content.Close()

		newFile, err := os.Create(newFilePath)
		if err != nil {
			return fmt.Errorf("creating output file %s: %w", newPath, err)
		}
		defer newFile.Close()

		_, err = io.Copy(newFile, content)
		if err != nil {
			return fmt.Errorf("writing file %s: %w", newPath, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("extracting archive %s: %w", src, err)
	}

	return nil
}
