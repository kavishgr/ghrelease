package utils

import (
	"debug/elf"
	"debug/macho"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// isBinary checks if a file is a valid executable binary for the current OS.
// Returns true if the file is a valid ELF (Linux) or Mach-O (macOS) binary.
func isBinary(path string, verifyFile func(*os.File) error) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	return verifyFile(f) == nil
}

// cleanup removes non-binary files from tempdir and moves all binaries to the root directory.
// It performs two passes:
//  1. Find all binaries in subdirectories and move them to tempdir root
//  2. Remove all non-binary files and empty subdirectories
func Cleanup(tempdir string) error {
	var verifyFile func(file *os.File) error

	switch runtime.GOOS {
	case "linux":
		verifyFile = func(file *os.File) error {
			_, err := elf.NewFile(file)
			return err
		}
	case "darwin":
		verifyFile = func(file *os.File) error {
			_, err := macho.NewFile(file)
			return err
		}
	}

	// FIRST PASS: Find all binaries and move them to tempdir root
	err := filepath.WalkDir(tempdir, func(binpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if binpath == tempdir {
			return nil
		}

		// Skip directories (we only care about files)
		if d.IsDir() {
			return nil
		}

		// Check if it's a valid binary
		if isBinary(binpath, verifyFile) {
			// It's a binary!
			// If it's in a subdirectory, move it to root
			if filepath.Dir(binpath) != tempdir {
				newPath := filepath.Join(tempdir, filepath.Base(binpath))

				// Handle filename conflicts
				if _, err := os.Stat(newPath); err == nil {
					// File already exists, add parent dir as prefix
					ext := filepath.Ext(filepath.Base(binpath))
					name := strings.TrimSuffix(filepath.Base(binpath), ext)
					parentDir := filepath.Base(filepath.Dir(binpath))
					newPath = filepath.Join(tempdir, name+"-"+parentDir+ext)
				}

				// Move the binary to root
				if err := os.Rename(binpath, newPath); err != nil {
					return err
				}
				binpath = newPath
			}

			// Make it executable
			if err := os.Chmod(binpath, 0755); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// SECOND PASS: Remove everything except binaries in root
	entries, err := os.ReadDir(tempdir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(tempdir, entry.Name())

		if entry.IsDir() {
			// Remove all subdirectories (binaries already moved out)
			if err := os.RemoveAll(path); err != nil {
				return err
			}
		} else {
			// For files in root, verify if they're binaries
			if !isBinary(path, verifyFile) {
				// Not a binary, remove it
				if err := os.Remove(path); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
