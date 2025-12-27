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
// extracted files. Returns an error if extraction fails or if a path traversal
// attempt is detected.
func Extractor(src, tempdir string) error {
	reader, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening file %s: %w", src, err)
	}
	defer reader.Close()

	ctx := context.Background()

	// Let the library identify the format automatically
	format, stream, err := archives.Identify(ctx, src, reader)
	if err != nil {
		// Not a recognized archive format, might be a plain binary
		// Check if file has no extension (plain binary)
		if strings.IndexByte(filepath.Base(src), '.') == -1 {
			return nil // Plain binary, nothing to extract
		}
		return fmt.Errorf("identifying archive format for %s: %w", src, err)
	}

	// Check if format supports extraction
	ex, ok := format.(archives.Extractor)
	if !ok {
		// Format identified but not extractable, might be plain file
		return nil
	}

	// Extract the archive
	err = ex.Extract(ctx, stream, func(ctx context.Context, f archives.FileInfo) error {
		// Strip the first directory component from the path
		pathInArchive := f.NameInArchive
		parts := strings.Split(filepath.ToSlash(pathInArchive), "/")

		var newPath string
		if len(parts) <= 1 {
			newPath = pathInArchive
		} else {
			newPath = filepath.Join(parts[1:]...)
		}

		// Skip if the path becomes empty (was just a top-level directory)
		if newPath == "" {
			return nil
		}

		newFilePath := filepath.Join(tempdir, filepath.Clean(newPath))

		// prevent path traversal
		cleanTempdir := filepath.Clean(tempdir)
		if !strings.HasPrefix(filepath.Clean(newFilePath), cleanTempdir) {
			return fmt.Errorf("invalid archive: path traversal attempt detected in %s", newPath)
		}

		// If it's a directory, just create it and return
		if f.IsDir() {
			return os.MkdirAll(newFilePath, 0755)
		}

		// For files, create parent directories first
		if err := os.MkdirAll(filepath.Dir(newFilePath), 0755); err != nil {
			return fmt.Errorf("creating parent dirs for %s: %w", newPath, err)
		}

		// Open the file content
		content, err := f.Open()
		if err != nil {
			return fmt.Errorf("opening file %s from archive: %w", newPath, err)
		}
		defer content.Close()

		// Create the output file
		newFile, err := os.Create(newFilePath)
		if err != nil {
			return fmt.Errorf("creating output file %s: %w", newPath, err)
		}
		defer newFile.Close()

		// Copy content
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
