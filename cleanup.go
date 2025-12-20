package main

import (
	"debug/elf"
	"debug/macho"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func cleanup(tempdir string) error {
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

		// Open and verify if it's a binary
		f, err := os.Open(binpath)
		if err != nil {
			return err
		}
		defer f.Close()

		// Check if it's a valid ELF or Mach-O binary
		if err := verifyFile(f); err == nil {
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
			// fmt.Println("Binary kept:", filepath.Base(binpath))
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
			os.RemoveAll(path)
			// fmt.Println("Removed directory:", entry.Name())
		} else {
			// For files in root, verify if they're binaries
			f, err := os.Open(path)
			if err != nil {
				continue
			}

			err = verifyFile(f)
			f.Close()

			if err != nil {
				// Not a binary, remove it
				os.Remove(path)
				// fmt.Println("Removed non-binary:", entry.Name())
			}
		}
	}

	return nil
}
