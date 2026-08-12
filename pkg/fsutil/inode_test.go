package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInodeKeyOf(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(a, []byte("x"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	b := filepath.Join(dir, "b.txt")
	if err := os.Link(a, b); err != nil {
		t.Fatalf("link: %v", err)
	}

	ia, okA := InodeKeyOf(mustStat(t, a))
	ib, okB := InodeKeyOf(mustStat(t, b))

	// на платформах без поддержки inode (Windows) хардлинки не распознаются —
	// это допустимое поведение, проверяем только консистентность
	if !okA {
		if okB {
			t.Fatal("inconsistent: a has no inode key but b does")
		}
		t.Skip("platform does not expose inode info")
	}

	if !okB {
		t.Fatal("hardlink b must resolve to an inode key when a does")
	}
	if ia.Ino != ib.Ino {
		t.Errorf("hardlinked files must share inode: %d vs %d", ia.Ino, ib.Ino)
	}
	if ia.Nlink < 2 {
		t.Errorf("expected nlink >= 2 for hardlinked file, got %d", ia.Nlink)
	}

	// обычный файл без хардлинков — отдельный inode, nlink 1
	c := filepath.Join(dir, "c.txt")
	if err := os.WriteFile(c, []byte("y"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	ic, okC := InodeKeyOf(mustStat(t, c))
	if !okC {
		t.Fatal("regular file must have inode key")
	}
	if ic.Ino == ia.Ino {
		t.Error("unrelated file must not share inode with hardlinked pair")
	}
}

func mustStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info
}
