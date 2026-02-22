package nanemu

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"debug/elf"
	"debug/pe"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const x86 = "x86_64"

func checkKernelArch(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	rd := bufio.NewReader(file)
	if isGzip(file) {
		gzr, _ := gzip.NewReader(file)
		defer gzr.Close()

		rd = bufio.NewReader(gzr)
	}

	b, err := io.ReadAll(io.LimitReader(rd, 128*1024))
	if err != nil {
		return "", fmt.Errorf("can't read kernel archive from %s: %w", path, err)
	}

	r := bytes.NewReader(b)

	buf := make([]byte, 512)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return "", nil
	}
	data := buf[:n]

	filename := strings.ToLower(filepath.Base(path))
	elfFile, err := elf.NewFile(r)
	if err == nil {
		switch elfFile.FileHeader.Machine {
		case elf.EM_X86_64:
			return "amd64", nil
		case elf.EM_AARCH64:
			return "arm64", nil
		}
	}

	peFile, err := pe.NewFile(r)
	if err == nil {
		switch peFile.FileHeader.Machine {
		case pe.IMAGE_FILE_MACHINE_AMD64:
			return "amd64", nil
		case pe.IMAGE_FILE_MACHINE_ARM64:
			return "arm64", nil
		}
	}

	if isBzImage(data) {
		return "amd64", nil
	}

	switch {
	case strings.Contains(filename, "arm64"),
		strings.Contains(filename, "aarch64"):
		return "arm64", nil
	case strings.Contains(filename, "amd64"),
		strings.Contains(filename, x86):
		return "amd64", nil
	}

	return "", nil
}

// isBzImage проверяет bzImage формат
func isBzImage(data []byte) bool {
	const minBzImageSize = 0x204
	if len(data) < minBzImageSize {
		return false
	}

	// Сигнатура setup header bzImage
	const setupSigOffset = 0x202
	return data[setupSigOffset] == 0x03 && data[setupSigOffset+1] == 0x00
}

// isGzip проверяет, является ли файл gzip-архивом
func isGzip(f *os.File) bool {
	buf := make([]byte, 2)
	_, err := f.ReadAt(buf, 0)
	if err != nil {
		return false
	}
	return buf[0] == 0x1F && buf[1] == 0x8B
}
