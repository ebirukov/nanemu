//go:build windows
// +build windows

package cpio

import (
	"os"
)

func inodeKey(info os.FileInfo) (fileKey, bool) {
	return fileKey{}, false
}
