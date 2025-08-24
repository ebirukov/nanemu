package text

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestLineMatchReader_Read(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		substr      string
		wantOutput  string
		expectMatch bool
		matchLine   string
	}{
		{
			name:        "single line with match",
			input:       "hello error world\n",
			substr:      "error",
			wantOutput:  "hello error world\n",
			expectMatch: true,
			matchLine:   "hello error world",
		},
		{
			name:        "single line without match",
			input:       "hello world\n",
			substr:      "error",
			wantOutput:  "hello world\n",
			expectMatch: false,
		},
		{
			name:        "multiple lines with one match",
			input:       "line1\nline with error\nline3\n",
			substr:      "error",
			wantOutput:  "line1\nline with error\nline3\n",
			expectMatch: true,
			matchLine:   "line with error",
		},
		{
			name:        "multiple matches",
			input:       "error1\nerror2\nnormal\n",
			substr:      "error",
			wantOutput:  "error1\nerror2\nnormal\n",
			expectMatch: true,
			// Будет два вызова onMatch
		},
		{
			name:        "empty lines",
			input:       "\n\nerror\n\n",
			substr:      "error",
			wantOutput:  "\n\nerror\n\n",
			expectMatch: true,
			matchLine:   "error",
		},
		{
			name:        "windows line endings",
			input:       "line with error\r\nnormal line\r\n",
			substr:      "error",
			wantOutput:  "line with error\r\nnormal line\r\n",
			expectMatch: true,
			matchLine:   "line with error",
		},
		{
			name:        "case sensitive match",
			input:       "Error\nERROR\nerror\n",
			substr:      "error",
			wantOutput:  "Error\nERROR\nerror\n",
			expectMatch: true,
			matchLine:   "error", // только точное совпадение
		},
		{
			name:        "partial word match",
			input:       "this is an error message\n",
			substr:      "err",
			wantOutput:  "this is an error message\n",
			expectMatch: true,
			matchLine:   "this is an error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var matched bool
			var matchedLine string

			// Создаем reader с callback
			reader := NewLineMatchReader(
				strings.NewReader(tt.input),
				tt.substr,
				func(line string) {
					matched = true
					matchedLine = line
				},
			)

			// Читаем все данные
			var output bytes.Buffer
			_, err := io.Copy(&output, reader)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Проверяем выходные данные
			if output.String() != tt.wantOutput {
				t.Errorf("Output mismatch.\nGot: %q\nWant: %q", output.String(), tt.wantOutput)
			}

			// Проверяем вызов callback
			if tt.expectMatch && !matched {
				t.Error("Expected match but callback was not called")
			}
			if !tt.expectMatch && matched {
				t.Error("Unexpected match callback")
			}

			// Проверяем переданную строку если ожидается конкретное значение
			if tt.matchLine != "" && matchedLine != tt.matchLine {
				t.Errorf("Match line mismatch. Got: %q, Want: %q", matchedLine, tt.matchLine)
			}
		})
	}
}

func TestLineMatchReader_Read_EdgeCases(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		var called bool
		reader := NewLineMatchReader(
			strings.NewReader(""),
			"test",
			func(line string) { called = true },
		)

		var buf [10]byte
		n, err := reader.Read(buf[:])
		if n != 0 || err != io.EOF {
			t.Errorf("Expected EOF, got n=%d, err=%v", n, err)
		}
		if called {
			t.Error("Callback should not be called for empty input")
		}
	})

	t.Run("partial read buffer", func(t *testing.T) {
		var matches []string
		reader := NewLineMatchReader(
			strings.NewReader("line with error\nanother line\n"),
			"error",
			func(line string) { matches = append(matches, line) },
		)

		// Читаем маленькими chunks
		buf := make([]byte, 5)
		var output bytes.Buffer
		for {
			n, err := reader.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			output.Write(buf[:n])
		}

		if len(matches) != 1 || matches[0] != "line with error" {
			t.Errorf("Expected one match 'line with error', got %v", matches)
		}
	})

	t.Run("multiple calls to small read", func(t *testing.T) {
		var matchCount int
		reader := NewLineMatchReader(
			strings.NewReader("error1\nerror2\nerror3\n"),
			"error",
			func(line string) { matchCount++ },
		)

		// Читаем по одному байту
		var output bytes.Buffer
		buf := make([]byte, 1)
		for {
			n, err := reader.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			output.Write(buf[:n])
		}

		if matchCount != 3 {
			t.Errorf("Expected 3 matches, got %d", matchCount)
		}
	})
}

func TestLineMatchReader_InterfaceCompatibility(t *testing.T) {
	// Тест на совместимость с io.Reader
	var _ io.Reader = (*LineMatchReader)(nil)

	// Тест что можно использовать с io.Copy
	input := strings.NewReader("test error line\n")
	var output bytes.Buffer
	var matched bool

	reader := NewLineMatchReader(input, "error", func(line string) {
		matched = true
	})

	n, err := io.Copy(&output, reader)
	if err != nil {
		t.Fatal(err)
	}

	if n != 16 { // "test error line\n" = 16 bytes
		t.Errorf("Expected 16 bytes copied, got %d", n)
	}
	if !matched {
		t.Error("Expected match callback")
	}
}
