package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// tempdir and opts.Dir
// remove tempdir afterwards
func moveBinaries(tempdir string, destinationDir string) error {
	if _, err := os.Stat(destinationDir); os.IsNotExist(err) {
		if err = os.MkdirAll(destinationDir, os.ModePerm); err != nil {
			return err
		}
	}
	files, err := os.ReadDir(tempdir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if !file.IsDir() {
			if err = os.Rename(filepath.Join(tempdir, file.Name()), filepath.Join(destinationDir, file.Name())); err != nil {
				return err
			}
		}
	}

	err = os.RemoveAll(tempdir)
	if err != nil{
		fmt.Printf("%s: not deleted...\n", tempdir)
	}
	return nil
}

