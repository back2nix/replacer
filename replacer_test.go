package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFunctionReplacer_extractFunctions(t *testing.T) {
	replacer := NewFunctionReplacer()

	tests := []struct {
		name     string
		filename string
		isGoFile bool
		expected []string
	}{
		{
			name:     "Go functions",
			filename: "testdata/source.go",
			isGoFile: true,
			expected: []string{"Hello", "Serve"},
		},
		{
			name:     "TypeScript functions",
			filename: "testdata/source.ts",
			isGoFile: false,
			expected: []string{"greet", "serve"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := readFile(tt.filename)
			if err != nil {
				t.Fatalf("Не удалось прочитать файл %s: %v", tt.filename, err)
			}

			// Добавляем отладочный вывод
			t.Logf("Содержимое файла %s:\n%s", tt.filename, content)

			functions, err := replacer.extractFunctions(content, tt.isGoFile)
			if err != nil {
				t.Fatalf("Ошибка извлечения функций: %v", err)
			}

			// Отладочный вывод найденных функций
			t.Logf("Найденные функции:")
			for i, fn := range functions {
				t.Logf("  [%d] Name: %s", i, fn.Name)
			}

			if len(functions) != len(tt.expected) {
				t.Errorf("Ожидалось %d функций, получено %d", len(tt.expected), len(functions))
			}

			for i, expected := range tt.expected {
				if i >= len(functions) {
					t.Errorf("Функция %s не найдена", expected)
					continue
				}
				if functions[i].Name != expected {
					t.Errorf("Ожидалось имя функции %s, получено %s", expected, functions[i].Name)
				}
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestFunctionReplacer_replaceFunctions_Go(t *testing.T) {
	replacer := NewFunctionReplacer()

	sourceContent, err := readFile("testdata/source.go")
	if err != nil {
		t.Fatalf("Не удалось прочитать source.go: %v", err)
	}

	targetContent, err := readFile("testdata/target.go")
	if err != nil {
		t.Fatalf("Не удалось прочитать target.go: %v", err)
	}

	sourceFunctions, err := replacer.extractFunctions(sourceContent, true)
	if err != nil {
		t.Fatalf("Ошибка извлечения функций из исходника: %v", err)
	}

	result := replacer.replaceFunctions(targetContent, sourceFunctions, true)

	// Проверяем, что функция Hello была заменена
	if !strings.Contains(result, "Hello from source!") {
		t.Error("Функция Hello не была заменена")
	}

	// Проверяем, что функция Bye осталась
	if !strings.Contains(result, "Goodbye from target!") {
		t.Error("Существующая функция Bye была удалена")
	}

	// Проверяем, что метод Serve был заменен
	if !strings.Contains(result, "Service is serving from source!") {
		t.Error("Метод Serve не был заменен")
	}

	// Проверяем, что существующий метод OldServe остался
	if !strings.Contains(result, "Old serving logic") {
		t.Error("Существующий метод OldServe был удален")
	}
}

func TestFunctionReplacer_replaceFunctions_TypeScript(t *testing.T) {
	replacer := NewFunctionReplacer()

	sourceContent, err := readFile("testdata/source.ts")
	if err != nil {
		t.Fatalf("Не удалось прочитать source.ts: %v", err)
	}

	targetContent, err := readFile("testdata/target.ts")
	if err != nil {
		t.Fatalf("Не удалось прочитать target.ts: %v", err)
	}

	sourceFunctions, err := replacer.extractFunctions(sourceContent, false)
	if err != nil {
		t.Fatalf("Ошибка извлечения функций из исходника: %v", err)
	}

	result := replacer.replaceFunctions(targetContent, sourceFunctions, false)

	// Проверяем, что функция greet была заменена
	if !strings.Contains(result, "Hello from source!") {
		t.Error("Функция greet не была заменена")
	}

	// Проверяем, что функция bye осталась
	if !strings.Contains(result, "Goodbye from target!") {
		t.Error("Существующая функция bye была удалена")
	}

	// Проверяем, что метод serve был заменен
	if !strings.Contains(result, "Serving from source!") {
		t.Error("Метод serve не был заменен")
	}

	// Проверяем, что существующий метод oldServe остался
	if !strings.Contains(result, "Old serving logic") {
		t.Error("Существующий метод oldServe был удален")
	}
}

func TestDetermineFileTypeFromContent(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool // true для Go, false для TypeScript
	}{
		{
			name:     "Go content",
			filename: "testdata/source.go",
			expected: true,
		},
		{
			name:     "TypeScript content",
			filename: "testdata/source.ts",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := readFile(tt.filename)
			if err != nil {
				t.Fatalf("Не удалось прочитать файл %s: %v", tt.filename, err)
			}

			result := determineFileTypeFromContent(content)
			if result != tt.expected {
				t.Errorf("Ожидался тип %v, получен %v", tt.expected, result)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedSource  string
		expectedTarget  string
		expectedClip    bool
		expectedValid   bool
	}{
		{
			name:            "Target only (clipboard source)",
			args:            []string{"target.go"},
			expectedSource:  "",
			expectedTarget:  "target.go",
			expectedClip:    true,
			expectedValid:   true,
		},
		{
			name:            "Source and target",
			args:            []string{"source.go", "target.go"},
			expectedSource:  "source.go",
			expectedTarget:  "target.go",
			expectedClip:    false,
			expectedValid:   true,
		},
		{
			name:            "With separator - clipboard",
			args:            []string{"--", "target.go"},
			expectedSource:  "",
			expectedTarget:  "target.go",
			expectedClip:    true,
			expectedValid:   true,
		},
		{
			name:            "With separator - files",
			args:            []string{"--", "source.go", "target.go"},
			expectedSource:  "source.go",
			expectedTarget:  "target.go",
			expectedClip:    false,
			expectedValid:   true,
		},
		{
			name:            "Invalid - only separator",
			args:            []string{"--"},
			expectedSource:  "",
			expectedTarget:  "",
			expectedClip:    false,
			expectedValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сохраняем оригинальные args
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			// Устанавливаем тестовые args
			os.Args = append([]string{"replacer"}, tt.args...)

			source, target, useClip, valid := parseArgs()

			if source != tt.expectedSource {
				t.Errorf("Ожидался source %s, получен %s", tt.expectedSource, source)
			}
			if target != tt.expectedTarget {
				t.Errorf("Ожидался target %s, получен %s", tt.expectedTarget, target)
			}
			if useClip != tt.expectedClip {
				t.Errorf("Ожидался clipboard %v, получен %v", tt.expectedClip, useClip)
			}
			if valid != tt.expectedValid {
				t.Errorf("Ожидалась валидность %v, получена %v", tt.expectedValid, valid)
			}
		})
	}
}

func TestEndToEnd_Integration(t *testing.T) {
	// Создаем временную директорию для выходных файлов
	tmpDir, err := os.MkdirTemp("", "replacer_test")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name       string
		sourceFile string
		targetFile string
		isGoFile   bool
		checks     []struct {
			shouldContain string
			description   string
		}
	}{
		{
			name:       "Go files integration",
			sourceFile: "testdata/source.go",
			targetFile: "testdata/target.go",
			isGoFile:   true,
			checks: []struct {
				shouldContain string
				description   string
			}{
				{"Hello from source!", "Функция Hello должна быть заменена"},
				{"Service is serving from source!", "Метод Serve должен быть заменен"},
				{"Goodbye from target!", "Функция Bye должна остаться"},
				{"Old serving logic", "Метод OldServe должен остаться"},
			},
		},
		{
			name:       "TypeScript files integration",
			sourceFile: "testdata/source.ts",
			targetFile: "testdata/target.ts",
			isGoFile:   false,
			checks: []struct {
				shouldContain string
				description   string
			}{
				{"Hello from source!", "Функция greet должна быть заменена"},
				{"Serving from source!", "Метод serve должен быть заменен"},
				{"Goodbye from target!", "Функция bye должна остаться"},
				{"Old serving logic", "Метод oldServe должен остаться"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replacer := NewFunctionReplacer()

			sourceContent, err := readFile(tt.sourceFile)
			if err != nil {
				t.Fatalf("Не удалось прочитать исходный файл %s: %v", tt.sourceFile, err)
			}

			targetContent, err := readFile(tt.targetFile)
			if err != nil {
				t.Fatalf("Не удалось прочитать целевой файл %s: %v", tt.targetFile, err)
			}

			sourceFunctions, err := replacer.extractFunctions(sourceContent, tt.isGoFile)
			if err != nil {
				t.Fatalf("Ошибка извлечения функций из исходника: %v", err)
			}

			result := replacer.replaceFunctions(targetContent, sourceFunctions, tt.isGoFile)

			// Проверяем все ожидаемые строки
			for _, check := range tt.checks {
				if !strings.Contains(result, check.shouldContain) {
					t.Errorf("%s: не найдена строка '%s'", check.description, check.shouldContain)
				}
			}

			// Опционально записываем результат в временный файл для отладки
			resultFile := filepath.Join(tmpDir, "result_"+filepath.Base(tt.targetFile))
			if err := os.WriteFile(resultFile, []byte(result), 0644); err != nil {
				t.Logf("Предупреждение: не удалось записать результат в %s: %v", resultFile, err)
			} else {
				t.Logf("Результат записан в %s", resultFile)
			}
		})
	}
}
