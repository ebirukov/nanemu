package text

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

// LineMatchReader реализует io.Reader и мониторит входящие строки на наличие подстрок.
// При обнаружении подстроки в строке вызывается callback-функция onMatch.
// Чтение происходит построчно, что гарантирует целостность обработки строк.
//
// Пример использования:
//
//	reader := NewLineMatchReader(input, "error", func(line string) {
//	    log.Printf("Found error in line: %s", line)
//	})
//	io.Copy(output, reader)
type LineMatchReader struct {
	br      *bufio.Reader
	buf     bytes.Buffer
	substr  string
	onMatch func(string)
}

// NewLineMatchReader создает новый LineMatchReader.
//
// Параметры:
//   - r: исходный io.Reader для чтения данных
//   - substr: подстрока для поиска в каждой строке
//   - onMatch: callback-функция, вызываемая при обнаружении подстроки
//
// Возвращает:
//   - *LineMatchReader: reader с функциональностью мониторинга
//
// Note: Reader не закрывает исходный io.Reader автоматически.
func NewLineMatchReader(r io.Reader, substr string, onMatch func(string)) *LineMatchReader {
	return &LineMatchReader{
		br:      bufio.NewReader(r),
		substr:  substr,
		onMatch: onMatch,
	}
}

// Read читает данные из underlying reader, мониторя наличие подстрок.
// Реализует интерфейс io.Reader.
//
// Параметры:
//   - p: байтовый буфер для записи данных
//
// Возвращает:
//   - int: количество прочитанных байтов
//   - error: ошибка чтения или io.EOF при завершении
func (lmr *LineMatchReader) Read(p []byte) (int, error) {
	// Если в буфере есть данные, возвращаем их first
	if lmr.buf.Len() > 0 {
		return lmr.buf.Read(p)
	}

	// Читаем следующую строку (включая \n или \r\n)
	line, err := lmr.br.ReadString('\n')
	if err != nil && len(line) == 0 {
		return 0, err
	}

	// Убираем терминальные символы строки для проверки
	trimmed := strings.TrimRight(line, "\r\n")

	// Проверяем наличие подстроки и вызываем callback при совпадении
	if strings.Contains(trimmed, lmr.substr) {
		lmr.onMatch(trimmed)
	}

	// Записываем оригинальную строку (с переводом) в буфер
	lmr.buf.WriteString(line)

	// Возвращаем данные из буфера
	return lmr.buf.Read(p)
}

// WriteTo позволяет писать напрямую, минуя промежуточный буфер (при использовании io.Copy)
func (lmr *LineMatchReader) WriteTo(w io.Writer) (int64, error) {
	var total int64
	for {
		line, err := lmr.br.ReadString('\n')
		if len(line) > 0 {
			trimmed := strings.TrimRight(line, "\r\n")
			if strings.Contains(trimmed, lmr.substr) {
				lmr.onMatch(trimmed)
			}
			n, werr := io.WriteString(w, line)
			total += int64(n)
			if werr != nil {
				return total, werr
			}
		}
		if err != nil {
			if err == io.EOF {
				return total, nil
			}
			return total, err
		}
	}
}
