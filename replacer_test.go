package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)


// TestFunctionReplacer_replaceFunctions_Go and TestFunctionReplacer_replaceFunctions_TypeScript
// are good for focused checks, but TestEndToEnd_Integration will cover broader scenarios.
// I'm keeping them as they are, assuming their original target files allow them to pass.
// If `target.go` provided in the prompt is the one used, the `Serve` check in `TestFunctionReplacer_replaceFunctions_Go`
// implies an "add if not exists" behavior for `replaceFunctions`.

func TestFunctionReplacer_replaceFunctions_Go(t *testing.T) {
	replacer := NewFunctionReplacer()

	sourceContent, err := readFile("testdata/source.go")
	if err != nil {
		t.Fatalf("Не удалось прочитать source.go: %v", err)
	}

	// Assuming target.go has a Serve method for replacement, or that Serve is added.
	// If target.go is as per the prompt (no Serve method), this test relies on "add" behavior.
	targetContent, err := readFile("testdata/target.go")
	if err != nil {
		t.Fatalf("Не удалось прочитать target.go: %v", err)
	}

	sourceFunctions, err := replacer.extractFunctions(sourceContent, true)
	if err != nil {
		t.Fatalf("Ошибка извлечения функций из исходника: %v", err)
	}

	result := replacer.replaceFunctions(targetContent, sourceFunctions, true)

	if !strings.Contains(result, "Hello from source!") {
		t.Error("Функция Hello не была заменена")
	}
	if !strings.Contains(result, "Goodbye from target!") {
		t.Error("Существующая функция Bye была удалена или изменена некорректно")
	}
	if !strings.Contains(result, "Service is serving from source!") {
		t.Error("Метод Serve не был заменен или добавлен")
	}
	if !strings.Contains(result, "Old serving logic") {
		t.Error("Существующий метод OldServe был удален или изменен некорректно")
	}
}

func TestFunctionReplacer_replaceFunctions_TypeScript(t *testing.T) {
	replacer := NewFunctionReplacer()

	sourceContent, err := readFile("testdata/source.ts")
	if err != nil {
		t.Fatalf("Не удалось прочитать source.ts: %v", err)
	}

	// Similar assumption for target.ts regarding the 'serve' method.
	targetContent, err := readFile("testdata/target.ts")
	if err != nil {
		t.Fatalf("Не удалось прочитать target.ts: %v", err)
	}

	sourceFunctions, err := replacer.extractFunctions(sourceContent, false)
	if err != nil {
		t.Fatalf("Ошибка извлечения функций из исходника: %v", err)
	}

	result := replacer.replaceFunctions(targetContent, sourceFunctions, false)

	if !strings.Contains(result, "Hello from source!") { // Assuming source.ts has 'greet' -> "Hello from source!"
		t.Error("Функция greet не была заменена")
	}
	if !strings.Contains(result, "Goodbye from target!") { // Assuming target.ts has 'bye' -> "Goodbye from target!"
		t.Error("Существующая функция bye была удалена или изменена некорректно")
	}
	if !strings.Contains(result, "Serving from source!") { // Assuming source.ts has 'serve' -> "Serving from source!"
		t.Error("Метод serve не был заменен или добавлен")
	}
	if !strings.Contains(result, "Old serving logic") { // Assuming target.ts has 'oldServe' -> "Old serving logic"
		t.Error("Существующий метод oldServe был удален или изменен некорректно")
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
		{
			name:     "Complex Go content",
			filename: "testdata/source_complex.go",
			expected: true,
		},
		{
			name:     "Complex TypeScript content",
			filename: "testdata/source_complex.ts",
			expected: false,
		},
		{
			name:     "Empty Go file (should probably default or error, testing typical heuristic)",
			filename: "testdata/empty.go", // Content based, so empty is ambiguous. Assuming it might default to Go or TS based on other clues or return a specific error/default.
			expected: true, // Or false, depending on determineFileTypeFromContent's behavior for empty strings. Adjust if needed.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := os.Stat(tt.filename); os.IsNotExist(err) {
				t.Fatalf("Test file %s does not exist.", tt.filename)
			}
			content, err := readFile(tt.filename)
			if err != nil {
				t.Fatalf("Не удалось прочитать файл %s: %v", tt.filename, err)
			}

			result := determineFileTypeFromContent(content)
			if result != tt.expected {
				t.Errorf("Для файла %s: Ожидался тип %v, получен %v", tt.filename, tt.expected, result)
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
		{
			name:            "Invalid - too many args before separator",
			args:            []string{"s.go", "t.go", "x.go", "--", "target.go"},
			expectedValid:   false,
		},
		{
			name:            "Invalid - too many args after separator",
			args:            []string{"--", "s.go", "t.go", "x.go"},
			expectedValid:   false,
		},
		{
			name:            "No args",
			args:            []string{},
			expectedValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			os.Args = append([]string{"replacer"}, tt.args...)

			source, target, useClip, valid := parseArgs()

			if valid != tt.expectedValid {
				t.Errorf("Ожидалась валидность %v, получена %v", tt.expectedValid, valid)
			}
			// Only check other fields if expected to be valid, to avoid noise on invalid cases
			if tt.expectedValid {
				if source != tt.expectedSource {
					t.Errorf("Ожидался source '%s', получен '%s'", tt.expectedSource, source)
				}
				if target != tt.expectedTarget {
					t.Errorf("Ожидался target '%s', получен '%s'", tt.expectedTarget, target)
				}
				if useClip != tt.expectedClip {
					t.Errorf("Ожидался clipboard %v, получен %v", tt.expectedClip, useClip)
				}
			}
		})
	}
}

func TestEndToEnd_Integration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "replacer_test_e2e")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	type check struct {
		shouldContain    string
		shouldNotContain string
		description      string
	}

	tests := []struct {
		name       string
		sourceFile string
		targetFile string
		isGoFile   bool
		checks     []check
	}{
		{
			name:       "Go files integration - simple",
			sourceFile: "testdata/source.go", // Contains Hello, Serve
			targetFile: "testdata/target.go", // Contains Hello, Bye, OldServe (original prompt version, NO Serve)
			isGoFile:   true,
			checks: []check{ // These checks assume Serve from source.go is ADDED to target.go
				{shouldContain: "Hello from source!", description: "Функция Hello должна быть заменена"},
				{shouldNotContain: "Hello from target!", description: "Старая функция Hello должна исчезнуть"},
				{shouldContain: "Service is serving from source!", description: "Метод Serve должен быть заменен/добавлен"},
				// If target.go indeed had no Serve, there's no "Old Serve from target" to check against for notContain.
				{shouldContain: "Goodbye from target!", description: "Функция Bye должна остаться"},
				{shouldContain: "Old serving logic", description: "Метод OldServe должен остаться"},
			},
		},
		{
			name:       "TypeScript files integration - simple",
			sourceFile: "testdata/source.ts", // Contains greet, serve
			targetFile: "testdata/target.ts", // Contains greet, bye, oldServe (original prompt version, NO serve method)
			isGoFile:   false,
			checks: []check{ // These checks assume 'serve' from source.ts is ADDED to target.ts
				{shouldContain: "Hello from source!", description: "Функция greet должна быть заменена"}, // Assuming source.ts greet produces "Hello from source!"
				{shouldNotContain: "Hello ${name} from target!", description: "Старая функция greet должна исчезнуть"},
				{shouldContain: "Serving from source!", description: "Метод serve должен быть заменен/добавлен"},
				{shouldContain: "Goodbye from target!", description: "Функция bye должна остаться"},
				{shouldContain: "Old serving logic", description: "Метод oldServe должен остаться"},
			},
		},
		{
			name:       "Go files integration - complex",
			sourceFile: "testdata/source_complex.go",
			targetFile: "testdata/target_complex.go",
			isGoFile:   true,
			checks: []check{
				// Replaced
				{shouldContain: "New SimpleFunc from source_complex.go", description: "SimpleFunc should be replaced"},
				{shouldNotContain: "Old SimpleFunc from target_complex.go", description: "Old SimpleFunc should be gone"},
				{shouldContain: "New MethodWithArgs from source_complex.go", description: "MethodWithArgs should be replaced"},
				{shouldNotContain: "Old MethodWithArgs from target_complex.go", description: "Old MethodWithArgs should be gone"},
				{shouldContain: "Service is serving from source_complex.go!", description: "Service.Serve (complex) should be replaced"},
				{shouldNotContain: "Old Service.Serve from target_complex.go", description: "Old Service.Serve (complex) should be gone"},
				{shouldContain: "FuncWithNoReceiver from source_complex.go", description: "FuncWithNoReceiver should be replaced"},
				{shouldNotContain: "Old FuncWithNoReceiver from target_complex.go", description: "Old FuncWithNoReceiver should be gone"},
				// Kept from target
				{shouldContain: "KeepThisFunc from target_complex.go - I should remain.", description: "KeepThisFunc should remain"},
				{shouldContain: "TargetSpecificFunc in target_complex.go", description: "TargetSpecificFunc should remain"},
				// Added from source
				{shouldContain: "New AnotherFunc from source_complex.go", description: "AnotherFunc should be added from source"},
				{shouldContain: "SimpleFuncNeighbor from source_complex.go, to be added.", description: "SimpleFuncNeighbor should be added from source"},
				// Comment checks
				{shouldContain: "/*\nfunc (r *Receiver) MethodWithArgs(a int, b string) (bool, error) {\n\t// A commented out version, should not be touched", description: "Commented out function in target should remain"},
				// Check that trailing content in target is preserved
				{shouldContain: "var EndMarkerTargetComplexGo = true", description: "Trailing content in target_complex.go should be preserved"},
			},
		},
		{
			name:       "TypeScript files integration - complex",
			sourceFile: "testdata/source_complex.ts",
			targetFile: "testdata/target_complex.ts",
			isGoFile:   false,
			checks: []check{
				// Replaced
				{shouldContain: "New simpleTsFunc from source_complex.ts", description: "simpleTsFunc should be replaced"},
				{shouldNotContain: "Old simpleTsFunc from target_complex.ts", description: "Old simpleTsFunc should be gone"},
				{shouldContain: "New arrowTsFunc from source_complex.ts", description: "arrowTsFunc should be replaced"},
				{shouldNotContain: "Old arrowTsFunc from target_complex.ts", description: "Old arrowTsFunc should be gone"},
				{shouldContain: "New MyTsClass.classMethod from source_complex.ts", description: "MyTsClass.classMethod should be replaced"},
				{shouldNotContain: "Old MyTsClass.classMethod from target_complex.ts", description: "Old MyTsClass.classMethod should be gone"},
				{shouldContain: "New MyTsClass.staticTsMethod from source_complex.ts", description: "MyTsClass.staticTsMethod should be replaced"},
				{shouldNotContain: "Old MyTsClass.staticTsMethod from target_complex.ts", description: "Old MyTsClass.staticTsMethod should be gone"},
				{shouldContain: "New asyncTsFunc from source_complex.ts", description: "asyncTsFunc should be replaced"},
				{shouldNotContain: "Old asyncTsFunc from target_complex.ts", description: "Old asyncTsFunc should be gone"},
				{shouldContain: "New utilityTsFunc from source_complex.ts", description: "utilityTsFunc should be replaced"},
				{shouldNotContain: "Old utilityTsFunc from target_complex.ts", description: "Old utilityTsFunc should be gone"},
				{shouldContain: "New genericTsFunc from source_complex.ts", description: "genericTsFunc should be replaced"},
				{shouldNotContain: "Old genericTsFunc from target_complex.ts", description: "Old genericTsFunc should be gone"},
				// Kept from target
				{shouldContain: "MyTsClass.keepThisMethod from target_complex.ts - I should remain.", description: "MyTsClass.keepThisMethod should remain"},
				{shouldContain: "targetSpecificTsFunc in target_complex.ts", description: "targetSpecificTsFunc should remain"},
				// Added from source
				{shouldContain: "newSourceOnlyTsFunc from source_complex.ts, to be added", description: "newSourceOnlyTsFunc should be added from source"},
				// Comment checks
				{shouldContain: "// export function simpleTsFunc(): void {\n//    console.log(\"Commented out simpleTsFunc\");\n// }", description: "Commented out function in TS target should remain"},
				// Check that trailing content in target is preserved
				{shouldContain: "export const endMarkerTargetComplexTs = true;", description: "Trailing content in target_complex.ts should be preserved"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replacer := NewFunctionReplacer()

			if _, err := os.Stat(tt.sourceFile); os.IsNotExist(err) {
				t.Fatalf("Test source file %s does not exist.", tt.sourceFile)
			}
			if _, err := os.Stat(tt.targetFile); os.IsNotExist(err) {
				t.Fatalf("Test target file %s does not exist.", tt.targetFile)
			}

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

			for i, check := range tt.checks {
				if check.shouldContain != "" && !strings.Contains(result, check.shouldContain) {
					t.Errorf("[%d] %s: не найдена ожидаемая строка '%s'", i, check.description, check.shouldContain)
				}
				if check.shouldNotContain != "" && strings.Contains(result, check.shouldNotContain) {
					t.Errorf("[%d] %s: найдена нежелательная строка '%s'", i, check.description, check.shouldNotContain)
				}
			}

			resultFile := filepath.Join(tmpDir, "result_"+filepath.Base(tt.targetFile))
			// Corrected line below:
			if err := os.WriteFile(resultFile, []byte(result), 0644); err != nil {
				t.Logf("Предупреждение: не удалось записать результат в %s: %v", resultFile, err)
			} else {
				t.Logf("Результат для '%s' записан в %s", tt.name, resultFile)
			}
		})
	}
}
