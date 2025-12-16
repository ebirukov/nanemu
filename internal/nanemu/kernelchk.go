package nanemu

import (
	"bytes"
	"encoding/binary"
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

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", nil
	}
	data := buf[:n]

	filename := strings.ToLower(filepath.Base(path))

	// Проверка форматов в порядке приоритета
	if arch := checkPECOFFArchitecture(data); arch != "" {
		return arch, nil
	}

	if arch := checkELFArchitecture(data); arch != "" {
		return arch, nil
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

// checkPECOFFArchitecture проверяет PE/COFF и возвращает архитектуру
func checkPECOFFArchitecture(data []byte) string {
	if !isValidPECOFF(data) {
		return ""
	}

	peOffset := binary.LittleEndian.Uint32(data[0x3C:0x40])
	machine := binary.LittleEndian.Uint16(data[peOffset+4 : peOffset+6])

	switch machine {
	case 0xAA64:
		return "arm64"
	case 0x8664:
		return "amd64"
	default:
		return ""
	}
}

// isValidPECOFF проверяет валидность PE/COFF заголовка
func isValidPECOFF(data []byte) bool {
	if len(data) < 64 {
		return false
	}

	// Проверка DOS заголовка
	if data[0] != 'M' || data[1] != 'Z' {
		return false
	}

	// Получение смещения PE заголовка
	peOffset := binary.LittleEndian.Uint32(data[0x3C:0x40])
	if int(peOffset)+4 >= len(data) {
		return false
	}

	// Проверка PE сигнатуры
	return bytes.Equal(data[peOffset:peOffset+4], []byte("PE\x00\x00"))
}

// checkELFArchitecture проверяет ELF и возвращает архитектуру
func checkELFArchitecture(data []byte) string {
	if !isValidELF(data) {
		return ""
	}

	switch binary.LittleEndian.Uint16(data[18:20]) {
	case 0xB7:
		return "arm64"
	case 0x3E:
		return "amd64"
	default:
		return ""
	}
}

// isValidELF проверяет валидность ELF заголовка
func isValidELF(data []byte) bool {
	if len(data) < 20 {
		return false
	}

	// Проверка ELF магической сигнатуры
	return data[0] == 0x7F &&
		data[1] == 'E' &&
		data[2] == 'L' &&
		data[3] == 'F'
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
