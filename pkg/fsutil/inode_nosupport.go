//go:build windows || plan9
// +build windows plan9

package fsutil

import "os"

// InodeKeyOf на платформах без поддержки inode (Windows, Plan 9) возвращает
// false — хардлинки не распознаются, поведение сводится к обычному копированию.
func InodeKeyOf(info os.FileInfo) (Inode, bool) {
	return Inode{}, false
}
