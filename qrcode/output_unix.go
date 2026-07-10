//go:build !windows

package qrcode

import "os"

func replaceFile(source, target string) error {
	return os.Rename(source, target)
}

func syncParentDirectory(directory string) error {
	file, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}
