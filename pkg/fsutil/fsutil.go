package fsutil

import (
	"bytes"
	"os"
)

// IsExecutableFile возвращает true, если файл — ELF-бинарник или скрипт
// (начинается с shebang). Используется, чтобы на платформах без надёжных битов
// исполнения (например Windows) проставлять exec-права утилитам в образе.
func IsExecutableFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, nil
	}

	header := make([]byte, 4)
	if !info.Mode().IsRegular() || info.Size() < int64(len(header)) {
		return false, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	if _, err := file.Read(header); err != nil {
		return false, err
	}

	switch {
	// ELF
	case bytes.Equal(header[:4], []byte{0x7F, 'E', 'L', 'F'}):
		return true, nil
	// script
	case bytes.Equal(header[:2], []byte{'#', '!'}):
		return true, nil
	default:
	}

	return false, nil
}
