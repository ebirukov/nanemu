//go:build !windows
// +build !windows

package cpio

import (
	"golang.org/x/sys/unix"
	"os"
)

func inodeKey(info os.FileInfo) (fileKey, bool) {
	sys := info.Sys()
	switch stat := sys.(type) {
	case *unix.Stat_t: // Unix, Linux, macOS
		return fileKey{Ino: int64(stat.Ino), Nlink: int(stat.Nlink)}, true
	default:
		return fileKey{}, false
	}
}
