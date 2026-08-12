//go:build !windows && !plan9
// +build !windows,!plan9

package fsutil

import (
	"os"
	"syscall"
)

// InodeKeyOf возвращает Inode для info, если платформа предоставляет inode.
// os.FileInfo.Sys() на Unix возвращает *syscall.Stat_t (а не *unix.Stat_t —
// это другой тип), поэтому ассертим именно его.
func InodeKeyOf(info os.FileInfo) (Inode, bool) {
	sys := info.Sys()
	switch stat := sys.(type) {
	case *syscall.Stat_t: // Unix, Linux, macOS
		return Inode{Ino: int64(stat.Ino), Nlink: int(stat.Nlink)}, true
	default:
		return Inode{}, false
	}
}
