package main

import (
	"os"
	"testing"
)

func TestFunctionReplacer_extractFunctions(t *testing.T) {
	replacer := NewFunctionReplacer()

	tests := []struct {
		name     string
		filename string
		isGoFile bool
		expected []string // Expected function names in order of appearance
	}{
		{
			name:     "User clipboard Go source - clean",
			filename: "testdata/user_clipboard_source.go",
			isGoFile: true,
			expected: []string{"RequestPremiumSession"},
		},
		{
			name:     "Go with mixed comments and unclosed block",
			filename: "testdata/go_mixed_comments.go",
			isGoFile: true,
			// NOTE: The file testdata/go_mixed_comments.go, in its original and current form,
			// has the critical "supposedly unclosed" comment block actually CLOSED with a "*/".
			// Therefore, PotentiallyAffectedByUnclosedComment and AnotherUnaffectedFunctionAfterPotentialClosure
			// ARE extracted by the current logic. The test expectation is updated to reflect this.
			expected: []string{"BlockCommentedFunc2", "RealFunction1", "PotentiallyAffectedByUnclosedComment", "AnotherUnaffectedFunctionAfterPotentialClosure", "FinalFunction"},
		},
		{
			name:     "Go functions - simple",
			filename: "testdata/source.go",
			isGoFile: true,
			expected: []string{"Hello", "Serve"},
		},
		{
			name:     "TypeScript functions - simple",
			filename: "testdata/source.ts",
			isGoFile: false,
			expected: []string{"greet", "serve"},
		},
		{
			name:     "Go functions - complex",
			filename: "testdata/source_complex.go",
			isGoFile: true,
			// Order: SimpleFunc, MethodWithArgs, AnotherFunc, SimpleFuncNeighbor, Serve, FuncWithNoReceiver
			expected: []string{"SimpleFunc", "MethodWithArgs", "AnotherFunc", "SimpleFuncNeighbor", "Serve", "FuncWithNoReceiver"},
		},
		{
			name:     "TypeScript functions - complex",
			filename: "testdata/source_complex.ts",
			isGoFile: false,
			// Order: simpleTsFunc, arrowTsFunc, classMethod, staticTsMethod, asyncTsFunc, utilityTsFunc, newSourceOnlyTsFunc, genericTsFunc
			expected: []string{"simpleTsFunc", "arrowTsFunc", "classMethod", "staticTsMethod", "asyncTsFunc", "utilityTsFunc", "newSourceOnlyTsFunc", "genericTsFunc"},
		},
		{
			name:     "Empty Go file",
			filename: "testdata/empty.go",
			isGoFile: true,
			expected: []string{},
		},
		{
			name:     "Empty TS file",
			filename: "testdata/empty.ts",
			isGoFile: false,
			expected: []string{},
		},
		{
			name:     "Go file with no functions",
			filename: "testdata/no_funcs.go",
			isGoFile: true,
			expected: []string{},
		},
		{
			name:     "TS file with no functions",
			filename: "testdata/no_funcs.ts",
			isGoFile: false,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure test files exist before reading
			if _, err := os.Stat(tt.filename); os.IsNotExist(err) {
				t.Fatalf("Test file %s does not exist. Please create it.", tt.filename)
			}

			content, err := readFile(tt.filename)
			if err != nil {
				t.Fatalf("Не удалось прочитать файл %s: %v", tt.filename, err)
			}

			t.Logf("Содержимое файла %s:\n%s", tt.filename, content)

			functions, err := replacer.extractFunctions(content, tt.isGoFile)
			if err != nil {
				t.Fatalf("Ошибка извлечения функций: %v", err)
			}

			t.Logf("Найденные функции (%d):", len(functions))
			extractedNames := make([]string, len(functions))
			for i, fn := range functions {
				t.Logf("  [%d] Name: %s", i, fn.Name)
				extractedNames[i] = fn.Name
			}

			if len(functions) != len(tt.expected) {
				t.Errorf("Ожидалось %d функций, получено %d. Expected: %v, Got: %v", len(tt.expected), len(functions), tt.expected, extractedNames)
			}

			// Check names and order
			limit := min(len(functions), len(tt.expected))
			for i := 0; i < limit; i++ {
				if functions[i].Name != tt.expected[i] {
					t.Errorf("Ожидалось имя функции [%d] %s, получено %s", i, tt.expected[i], functions[i].Name)
				}
			}
		})
	}
}
