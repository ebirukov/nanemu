package nanemu

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"debug/elf"
	"debug/pe"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var ErrUnsupportedKernel = errors.New("unsupported kernel format")

func findKernel(path string) (string, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", "", fmt.Errorf("can't stat kernel path %s: %w", path, err)
	}

	if info.Mode().IsRegular() {
		arch, err := checkKernelArch(path)
		if err != nil {
			return "", "", err
		}

		return path, arch, nil
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return "", "", fmt.Errorf("can't read directory %s: %w", path, err)
	}

	for _, file := range files {
		path = filepath.Join(path, file.Name())
		arch, err := checkKernelArch(path)
		if err == nil {
			return path, arch, nil
		}
	}

	return "", "", fmt.Errorf("can't find kernel on path %s: %w", path, os.ErrNotExist)
}

func checkKernelArch(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	rd := bufio.NewReader(file)
	if isGzip(file) {
		gzr, err := gzip.NewReader(file)
		if err != nil {
			return "", fmt.Errorf("can't open kernel archive %s: %w", path, err)
		}

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
		return "", err
	}
	data := buf[:n]
	elfFile, err := elf.NewFile(r)
	if err == nil {
		for _, sec := range elfFile.Sections {
			if sec.Name == ".note.Linux" {
				switch elfFile.FileHeader.Machine {
				case elf.EM_X86_64:
					return "amd64", nil
				case elf.EM_AARCH64:
					return "arm64", nil
				}
			}
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

	return "", ErrUnsupportedKernel
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
