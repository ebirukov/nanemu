package tar

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// entry описывает одну запись в tar-архиве.
type entry struct {
	name     string
	typeflag byte
	mode     int64
	data     []byte
	linkname string
}

// buildTar собирает tar-архив в памяти из набора записей.
func buildTar(t *testing.T, entries []entry) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	defer tw.Close()

	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Typeflag: e.typeflag,
			Mode:     e.mode,
		}
		switch e.typeflag {
		case tar.TypeReg:
			hdr.Size = int64(len(e.data))
		case tar.TypeSymlink, tar.TypeLink:
			hdr.Linkname = e.linkname
		}

		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header %q: %v", e.name, err)
		}
		if e.typeflag == tar.TypeReg {
			if _, err := tw.Write(e.data); err != nil {
				t.Fatalf("write data %q: %v", e.name, err)
			}
		}
	}

	return &buf
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestUnpackTo(t *testing.T) {
	tests := []struct {
		name    string
		entries []entry
		wantErr bool
		check   func(t *testing.T, dir string)
	}{
		{
			name: "regular file",
			entries: []entry{
				{name: "file.txt", typeflag: tar.TypeReg, mode: 0644, data: []byte("hello")},
			},
			check: func(t *testing.T, dir string) {
				if got := readFile(t, filepath.Join(dir, "file.txt")); got != "hello" {
					t.Errorf("content = %q, want %q", got, "hello")
				}
			},
		},
		{
			name: "nested directory with file",
			entries: []entry{
				{name: "sub", typeflag: tar.TypeDir, mode: 0755},
				{name: "sub/inner.txt", typeflag: tar.TypeReg, mode: 0644, data: []byte("x")},
			},
			check: func(t *testing.T, dir string) {
				if got := readFile(t, filepath.Join(dir, "sub", "inner.txt")); got != "x" {
					t.Errorf("content = %q, want %q", got, "x")
				}
			},
		},
		{
			name: "symlink",
			entries: []entry{
				{name: "target.txt", typeflag: tar.TypeReg, mode: 0644, data: []byte("t")},
				{name: "link", typeflag: tar.TypeSymlink, mode: 0777, linkname: "target.txt"},
			},
			check: func(t *testing.T, dir string) {
				if runtime.GOOS == "windows" {
					t.Skip("symlinks are converted to hardlinks on windows")
				}
				got, err := os.Readlink(filepath.Join(dir, "link"))
				if err != nil {
					t.Fatalf("readlink: %v", err)
				}
				if got != "target.txt" {
					t.Errorf("symlink target = %q, want %q", got, "target.txt")
				}
			},
		},
		{
			name: "hardlink target before link",
			entries: []entry{
				{name: "a.txt", typeflag: tar.TypeReg, mode: 0644, data: []byte("content")},
				{name: "b.txt", typeflag: tar.TypeLink, linkname: "a.txt"},
			},
			check: func(t *testing.T, dir string) {
				if readFile(t, filepath.Join(dir, "b.txt")) != "content" {
					t.Error("hardlink b.txt does not resolve to a.txt content")
				}
				if !sameFile(t, filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")) {
					t.Error("a.txt and b.txt are not the same inode (hardlink not preserved)")
				}
			},
		},
		{
			name: "hardlink link before target",
			entries: []entry{
				{name: "b.txt", typeflag: tar.TypeLink, linkname: "a.txt"},
				{name: "a.txt", typeflag: tar.TypeReg, mode: 0644, data: []byte("content")},
			},
			check: func(t *testing.T, dir string) {
				if readFile(t, filepath.Join(dir, "b.txt")) != "content" {
					t.Error("hardlink b.txt does not resolve to a.txt content")
				}
				if !sameFile(t, filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")) {
					t.Error("a.txt and b.txt are not the same inode (hardlink not preserved)")
				}
			},
		},
		{
			name: "path traversal rejected",
			entries: []entry{
				{name: "../evil.txt", typeflag: tar.TypeReg, mode: 0644, data: []byte("pwn")},
			},
			wantErr: true,
		},
		{
			name: "unsupported entry skipped",
			entries: []entry{
				{name: "dev/zero", typeflag: tar.TypeChar, mode: 0666},
			},
			check: func(t *testing.T, dir string) {
				if _, err := os.Stat(filepath.Join(dir, "dev", "zero")); !os.IsNotExist(err) {
					t.Error("unsupported entry should not be created on disk")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			archive := buildTar(t, tt.entries)

			err := UnpackTo(archive, dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnpackTo() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.check != nil {
				tt.check(t, dir)
			}
		})
	}
}

func sameFile(t *testing.T, a, b string) bool {
	t.Helper()
	ai, err := os.Stat(a)
	if err != nil {
		t.Fatalf("stat %s: %v", a, err)
	}
	bi, err := os.Stat(b)
	if err != nil {
		t.Fatalf("stat %s: %v", b, err)
	}
	return os.SameFile(ai, bi)
}
