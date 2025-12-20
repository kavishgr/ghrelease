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

func Extractor(src, tempdir string) error {
	supportFormat := map[string]bool{
		// Compression formats
		".br":   true, // brotli
		".bz2":  true, // bzip2
		".gz":   true, // gzip
		".gzip": true, // gzip alternative extension
		".lz4":  true, // lz4
		".lz":   true, // lzip
		".mz":   true, // minlz
		".sz":   true, // snappy
		".s2":   true, // S2
		".xz":   true, // xz
		".zz":   true, // zlib
		".zst":  true, // zstandard
		// Archive formats
		".zip": true,
		".tar": true,
		".rar": true, // read-only
		".7z":  true, // read-only
		// Compressed tar variants
		".tar.gz":  true,
		".tar.bz2": true,
		".tar.xz":  true,
		".tar.lz4": true,
		".tar.zst": true,
		".tar.br":  true,
		".tar.sz":  true,
		".tar.s2":  true,
		".tgz":     true, // shorthand for .tar.gz
		".tbz":     true, // shorthand for .tar.bz2
		".tbz2":    true, // alternative for .tar.bz2
		".txz":     true, // shorthand for .tar.xz
	}

	// Check if format is supported
	var isSupported = false
	for format := range supportFormat {
		if strings.HasSuffix(src, format) {
			isSupported = true
			break
		}
	}

	if !isSupported {
		// check if the file has no suffix at all
		if strings.IndexByte(src, '.') == -1 {
			// just a binary, not an archive, or compressed archive
			return nil
		}
		return fmt.Errorf("%s is not supported", src)
	}

	reader, err := os.Open(src)
	if err != nil {
		return err
	}
	defer reader.Close()

	ctx := context.Background()

	// Identify returns format, stream, error
	format, stream, err := archives.Identify(ctx, src, reader)
	if err != nil {
		return err
	}

	// Check if format supports extraction
	if ex, ok := format.(archives.Extractor); ok {
		err = ex.Extract(ctx, stream, func(ctx context.Context, f archives.FileInfo) error {
			// Strip the first directory component from the path
			pathInArchive := f.NameInArchive
			parts := strings.Split(filepath.ToSlash(pathInArchive), "/")

			// If there's only one part (root-level file), keep it as is
			// Otherwise, strip the first directory
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

			newFilePath := filepath.Join(tempdir, newPath)

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
				return fmt.Errorf("opening %s: %w", newPath, err)
			}
			defer content.Close()

			// Create the output file
			newFile, err := os.Create(newFilePath)
			if err != nil {
				return fmt.Errorf("creating %s: %w", newPath, err)
			}
			defer newFile.Close()

			// Copy content
			_, err = io.Copy(newFile, content)
			if err != nil {
				return fmt.Errorf("copying %s: %w", newPath, err)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}
