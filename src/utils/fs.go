package utils

import (
	"os"
	"path"
)

func FileOK(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func FileName(p string) string {
	return path.Base(p)
}