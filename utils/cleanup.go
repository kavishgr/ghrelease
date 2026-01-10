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

func isBinary(path string, verifyFile func(*os.File) error) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	return verifyFile(f) == nil
}

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

	err := filepath.WalkDir(tempdir, func(binpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if binpath == tempdir {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if isBinary(binpath, verifyFile) {
			if filepath.Dir(binpath) != tempdir {
				newPath := filepath.Join(tempdir, filepath.Base(binpath))

				if _, err := os.Stat(newPath); err == nil {
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

			if err := os.Chmod(binpath, 0755); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	entries, err := os.ReadDir(tempdir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(tempdir, entry.Name())
		filename := entry.Name()

		// Skip the error log
		if filename == "error.log" {
			continue
		}

		if entry.IsDir() {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
		} else {
			if !isBinary(path, verifyFile) {
				if err := os.Remove(path); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
