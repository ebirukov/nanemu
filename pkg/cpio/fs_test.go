package cpio

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/cavaliergopher/cpio"
)

func TestCreate(t *testing.T) {
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	scriptFile := filepath.Join(tmpDir, "script.sh")
	if err := os.WriteFile(scriptFile, []byte("#!/bin/sh\necho test"), 0o644); err != nil {
		t.Fatalf("failed to create script file: %v", err)
	}

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	subFile := filepath.Join(subDir, "sub.txt")
	if err := os.WriteFile(subFile, []byte("subdata"), 0o644); err != nil {
		t.Fatalf("failed to create sub file: %v", err)
	}

	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.Mkdir(emptyDir, 0o755); err != nil {
		t.Fatalf("failed to create empty dir: %v", err)
	}

	tests := []struct {
		name         string
		rootPath     string
		execPermBits int
		wantErr      bool
		check        func(t *testing.T, buf *bytes.Buffer)
	}{
		{
			name:     "valid directory with files, no execPermBits",
			rootPath: tmpDir,
			wantErr:  false,
			check: func(t *testing.T, buf *bytes.Buffer) {
				r := cpio.NewReader(buf)
				modes := map[string]cpio.FileMode{}
				for {
					h, err := r.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatalf("unexpected error reading archive: %v", err)
					}
					modes[h.Name] = h.Mode
				}
				if got := modes["script.sh"].Perm(); got&0o111 != 0 {
					t.Errorf("expected script.sh to have no execute bits, got mode %v", got)
				}
			},
		},
		{
			name:         "valid directory with execPermBits set",
			rootPath:     tmpDir,
			execPermBits: 0o110, // u+x, g+x
			wantErr:      false,
			check: func(t *testing.T, buf *bytes.Buffer) {
				r := cpio.NewReader(buf)
				modes := map[string]cpio.FileMode{}
				for {
					h, err := r.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatalf("unexpected error reading archive: %v", err)
					}
					modes[h.Name] = h.Mode
				}
				got := modes["script.sh"]
				if got&0o110 != 0o110 {
					t.Errorf("expected script.sh to have exec bits 0o110, got %04o", got)
				}
			},
		},
		{
			name:     "empty directory only",
			rootPath: emptyDir,
			wantErr:  false,
			check: func(t *testing.T, buf *bytes.Buffer) {
				r := cpio.NewReader(buf)
				h, err := r.Next()
				if err != nil && err != io.EOF {
					t.Fatalf("unexpected error reading archive: %v", err)
				}
				if h != nil && h.Name != "." && h.Name != "" {
					t.Errorf("expected no files in empty dir archive, got %q", h.Name)
				}
			},
		},
		{
			name:         "single file instead of directory",
			rootPath:     filePath,
			execPermBits: 0,
			wantErr:      false,
			check: func(t *testing.T, buf *bytes.Buffer) {
				r := cpio.NewReader(buf)
				h, err := r.Next()
				if err != nil {
					t.Fatalf("unexpected error reading archive: %v", err)
				}
				if h.Name != filepath.Base(filePath) {
					t.Errorf("expected single file name %q, got %q", filepath.Base(filePath), h.Name)
				}
				if _, err := r.Next(); err != io.EOF {
					t.Errorf("expected only one entry in archive")
				}
			},
		},
		{
			name:         "nonexistent root",
			rootPath:     filepath.Join(tmpDir, "doesnotexist"),
			execPermBits: 0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := Create(&buf, tt.rootPath, tt.execPermBits)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.check != nil {
				tt.check(t, &buf)
			}
		})
	}
}
