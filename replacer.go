package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
)

type Function struct {
	Name     string
	Receiver string // Go-specific
	FullText string
	StartPos int // For sorting and potentially more robust deduplication
}

type FunctionReplacer struct {
}

func NewFunctionReplacer() *FunctionReplacer {
	return &FunctionReplacer{}
}

// Helper to check if a function match is commented out
func isMatchCommented(content string, matchStartIndex int) bool {
	// Check for single-line comment on the same line before the match
	lineStart := 0
	if idx := strings.LastIndex(content[:matchStartIndex], "\n"); idx != -1 {
		lineStart = idx + 1
	}

	lineContentBeforeMatch := content[lineStart:matchStartIndex]
	trimmedLineContentBeforeMatch := strings.TrimSpace(lineContentBeforeMatch)

	if strings.HasPrefix(trimmedLineContentBeforeMatch, "//") {
		return true
	}

	// Check for block comment /* ... */ by scanning content before the match
	openComment := "/*"
	closeComment := "*/"
	inBlockComment := false
	searchArea := content[:matchStartIndex]

	idx := 0
	for idx < len(searchArea) {
		isOpen := false
		isClose := false
		if !inBlockComment && idx+len(openComment) <= len(searchArea) && searchArea[idx:idx+len(openComment)] == openComment {
			isOpen = true
		} else if inBlockComment && idx+len(closeComment) <= len(searchArea) && searchArea[idx:idx+len(closeComment)] == closeComment {
			isClose = true
		}

		if isOpen {
			inBlockComment = true
			idx += len(openComment)
		} else if isClose {
			inBlockComment = false
			idx += len(closeComment)
		} else {
			idx++
		}
	}
	return inBlockComment
}

// min helper for logging snippets
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractGoFunctionWithBraceBalancing extracts a Go function using brace balancing approach
func extractGoFunctionWithBraceBalancing(content string, startIndex int) (string, int, error) {
	// Find the opening brace
	braceIndex := -1
	for i := startIndex; i < len(content); i++ {
		if content[i] == '{' {
			braceIndex = i
			break
		}
		// If we encounter a newline without finding an opening brace and there's
		// no "=" or other continuation, this might not be a function definition
		if content[i] == '\n' {
			line := strings.TrimSpace(content[startIndex:i])
			if !strings.Contains(line, "=") && !strings.HasSuffix(line, ",") && !strings.HasSuffix(line, "(") {
				return "", -1, fmt.Errorf("no opening brace found")
			}
		}
	}

	if braceIndex == -1 {
		return "", -1, fmt.Errorf("no opening brace found")
	}

	// Now balance braces
	braceCount := 1
	endIndex := braceIndex + 1

	for endIndex < len(content) && braceCount > 0 {
		switch content[endIndex] {
		case '{':
			braceCount++
		case '}':
			braceCount--
		case '"':
			// Skip string literals
			endIndex++
			for endIndex < len(content) && content[endIndex] != '"' {
				if content[endIndex] == '\\' {
					endIndex++ // Skip escaped character
				}
				endIndex++
			}
		case '\'':
			// Skip character literals
			endIndex++
			for endIndex < len(content) && content[endIndex] != '\'' {
				if content[endIndex] == '\\' {
					endIndex++ // Skip escaped character
				}
				endIndex++
			}
		case '/':
			// Skip comments
			if endIndex+1 < len(content) {
				if content[endIndex+1] == '/' {
					// Single-line comment
					for endIndex < len(content) && content[endIndex] != '\n' {
						endIndex++
					}
					continue
				} else if content[endIndex+1] == '*' {
					// Multi-line comment
					endIndex += 2
					for endIndex+1 < len(content) {
						if content[endIndex] == '*' && content[endIndex+1] == '/' {
							endIndex += 2
							break
						}
						endIndex++
					}
					continue
				}
			}
		}
		endIndex++
	}

	if braceCount != 0 {
		return "", -1, fmt.Errorf("unbalanced braces")
	}

	return content[startIndex:endIndex], endIndex, nil
}

func (fr *FunctionReplacer) extractFunctions(content string, isGoFile bool) ([]Function, error) {
	var functions []Function

	if isGoFile {
		isPotentiallyProblematic := strings.Contains(content, "RequestPremiumSession")

		if isPotentiallyProblematic {
			log.Printf("[EXTRACT_GO_DEBUG] Processing Go content containing 'RequestPremiumSession'. Content length: %d", len(content))
			// Выводим первые N символов для проверки на мусор в начале строки
			log.Printf("[EXTRACT_GO_DEBUG] Content Snippet (first 500 chars):\n<<<<SNIPPET_START>>>>\n%s\n<<<<SNIPPET_END>>>>", content[:min(500, len(content))])
		}

		// Use a simpler regex to find function headers, then use brace balancing for body
		funcHeaderRegexStr := `func\s*(?:\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\s*\(`
		funcHeaderRegex := regexp.MustCompile(funcHeaderRegexStr)

		if isPotentiallyProblematic {
			log.Printf("[EXTRACT_GO_DEBUG] Using Header Regex: %s", funcHeaderRegexStr)
		}

		matchesIndices := funcHeaderRegex.FindAllStringSubmatchIndex(content, -1)

		if isPotentiallyProblematic {
			log.Printf("[EXTRACT_GO_DEBUG] Found %d potential header matches with this regex.", len(matchesIndices))
		}

		for i, matchIdx := range matchesIndices {
			if len(matchIdx) < 4 {
				if isPotentiallyProblematic {
					log.Printf("[EXTRACT_GO_DEBUG] Match %d has insufficient indices: %v. Skipping.", i, matchIdx)
				}
				continue
			}

			funcName := content[matchIdx[2]:matchIdx[3]]
			matchStartIndexInContent := matchIdx[0]

			if isPotentiallyProblematic {
				log.Printf("[EXTRACT_GO_DEBUG] Potential Match %d: Name='%s', StartIndex=%d", i, funcName, matchStartIndexInContent)
			}

			commented := isMatchCommented(content, matchStartIndexInContent)
			if isPotentiallyProblematic {
				log.Printf("[EXTRACT_GO_DEBUG] Potential Match %d: Name='%s', IsCommented: %v", i, funcName, commented)
			}

			if commented {
				continue
			}

			// Extract full function using brace balancing
			fullFunctionText, endIndex, err := extractGoFunctionWithBraceBalancing(content, matchStartIndexInContent)
			if err != nil {
				if isPotentiallyProblematic {
					log.Printf("[EXTRACT_GO_DEBUG] Failed to extract function %s: %v", funcName, err)
				}
				continue
			}

			receiver := extractGoReceiver(fullFunctionText)
			if isPotentiallyProblematic {
				log.Printf("[EXTRACT_GO_DEBUG] Match %d: Name='%s', Receiver: '%s', EndIndex: %d. Adding to results.", i, funcName, receiver, endIndex)
			}

			functions = append(functions, Function{
				Name:     funcName,
				Receiver: receiver,
				FullText: strings.TrimSpace(fullFunctionText),
				StartPos: matchStartIndexInContent,
			})
		}
		if isPotentiallyProblematic {
			log.Printf("[EXTRACT_GO_DEBUG] Finished processing. Extracted %d functions for this content.", len(functions))
			for idx, f := range functions {
				log.Printf("[EXTRACT_GO_DEBUG] Result %d: Name: %s", idx, f.Name)
			}
		}

	} else { // TypeScript
		tempFunctions := []Function{}

		regexes := []struct {
			regex    *regexp.Regexp
			isMethod bool
		}{
			{
				regex:    regexp.MustCompile(`(?s)((?:export\s+)?(?:async\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)(?:\s*<[^>]+>)?\s*\((?:[^)]|\n)*\)\s*(?::\s*[^\{]+)?{((?:[^{}]|{[^{}]*})*)})`),
				isMethod: false,
			},
			{
				regex:    regexp.MustCompile(`(?s)((?:export\s+)?(?:const|let|var)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:async\s+)?(?:<[^>]+>)?\s*\((?:[^)]|\n)*\)\s*(?::\s*[^=]+)?\s*=>\s*{((?:[^{}]|{[^{}]*})*)})`),
				isMethod: false,
			},
			{
				regex:    regexp.MustCompile(`(?m)^([ \t]*(?:(?:public|private|protected|static|async)\s+)?([A-Za-z_][A-Za-z0-9_]*)(?:\s*<[^>]+>)?\s*\((?:[^)]|\n)*\)\s*(?::\s*[^\{]+)?{((?:[^{}]|{[^{}]*})*)})`),
				isMethod: true,
			},
		}

		for _, r := range regexes {
			matches := r.regex.FindAllStringSubmatchIndex(content, -1)
			for _, matchIdx := range matches {
				fullMatchText := content[matchIdx[0]:matchIdx[1]]
				funcName := content[matchIdx[4]:matchIdx[5]]
				matchStartIndex := matchIdx[0]

				if isMatchCommented(content, matchStartIndex) {
					continue
				}

				if r.isMethod && funcName == "constructor" {
					continue
				}

				tempFunctions = append(tempFunctions, Function{
					Name:     funcName,
					FullText: strings.TrimSpace(fullMatchText),
					StartPos: matchStartIndex,
				})
			}
		}

		sort.SliceStable(tempFunctions, func(i, j int) bool {
			return tempFunctions[i].StartPos < tempFunctions[j].StartPos
		})

		seenStartPos := make(map[int]bool)
		for _, fn := range tempFunctions {
			if !seenStartPos[fn.StartPos] {
				functions = append(functions, fn)
				seenStartPos[fn.StartPos] = true
			}
		}
	}

	return functions, nil
}

func extractGoReceiver(funcText string) string {
	receiverRegexStr := `func\s*\(([^)]*)\)\s*[A-Za-z_][A-Za-z0-9_]*\s*\(`
	receiverRegex := regexp.MustCompile(receiverRegexStr)
	matches := receiverRegex.FindStringSubmatch(funcText)
	if len(matches) > 1 {
		receiverPart := strings.TrimSpace(matches[1])
		return receiverPart
	}
	return ""
}

func (fr *FunctionReplacer) replaceFunctions(targetContent string, sourceFunctions []Function, isGoFile bool) string {
	result := targetContent

	targetFunctions, err := fr.extractFunctions(targetContent, isGoFile)
	if err != nil {
		log.Printf("Предупреждение: ошибка при парсинге целевого файла для существующих функций: %v", err)
	}

	targetFuncMap := make(map[string]Function)
	for _, fn := range targetFunctions {
		key := fr.getFunctionKey(fn, isGoFile)
		targetFuncMap[key] = fn
	}

	processedTargetKeys := make(map[string]bool)
	var newFunctionsToAdd []Function

	for _, sourceFn := range sourceFunctions {
		key := fr.getFunctionKey(sourceFn, isGoFile)
		if targetFn, exists := targetFuncMap[key]; exists {
			if !processedTargetKeys[key] {
				if strings.TrimSpace(targetFn.FullText) != "" && strings.Contains(result, targetFn.FullText) {
					result = strings.Replace(result, targetFn.FullText, sourceFn.FullText, 1)
				} else if strings.TrimSpace(targetFn.FullText) == "" {
                    log.Printf("Warning: Key '%s' matched for source func '%s', but target FullText was empty. Skipping replacement. Source FullText: `%s`", key, sourceFn.Name, sourceFn.FullText)
                } else {
					log.Printf("Warning: Key '%s' matched for source func '%s', but target FullText (`%s`) was not found in current result for replacement. Target might have been modified or FullText extraction differs. Source FullText: `%s`", key, sourceFn.Name, targetFn.FullText, sourceFn.FullText)
				}
				processedTargetKeys[key] = true
			}
		} else {
			newFunctionsToAdd = append(newFunctionsToAdd, sourceFn)
		}
	}

	if len(newFunctionsToAdd) > 0 {
		sb := strings.Builder{}
		sb.WriteString(result)

		if len(strings.TrimSpace(result)) > 0 && !strings.HasSuffix(result, "\n") {
			sb.WriteString("\n")
		}

		for i, sourceFnToAdd := range newFunctionsToAdd {
			currentResultString := sb.String()
			if len(strings.TrimSpace(currentResultString)) > 0 {
				if !strings.HasSuffix(currentResultString, "\n\n") && !strings.HasSuffix(currentResultString, "\n\n\n") {
					if strings.HasSuffix(currentResultString, "\n") {
						sb.WriteString("\n")
					} else {
						sb.WriteString("\n\n")
					}
				}
			} else if i > 0 {
                 sb.WriteString("\n\n")
            }


			sb.WriteString(sourceFnToAdd.FullText)
			sb.WriteString("\n")
		}
		result = sb.String()
	}
	return result
}

func (fr *FunctionReplacer) getFunctionKey(fn Function, isGoFile bool) string {
	if isGoFile && fn.Receiver != "" {
		receiverParts := strings.Fields(fn.Receiver)
		var receiverType string
		if len(receiverParts) > 0 {
			receiverType = strings.TrimPrefix(receiverParts[len(receiverParts)-1], "*")
		}
		if receiverType != "" {
			return fmt.Sprintf("%s.%s", receiverType, fn.Name)
		}
		return fmt.Sprintf("receiver_%s.%s", strings.ReplaceAll(fn.Receiver, " ", "_"), fn.Name)

	}
	return fn.Name
}

func readFile(filename string) (string, error) {
	contentBytes, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("не удалось прочитать файл %s: %v", filename, err)
	}
	return string(contentBytes), nil
}

func readFromClipboard() (string, error) {
	content, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("не удалось прочитать из буфера обмена: %v", err)
	}
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("буфер обмена пуст")
	}
	return content, nil
}

func writeFile(filename, content string) error {
	if strings.TrimSpace(content) != "" {
		normalizedContent := strings.ReplaceAll(content, "\r\n", "\n")
		trimmedContent := strings.TrimRight(normalizedContent, "\n")
		content = trimmedContent + "\n"
	} else {
		content = ""
	}

	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("не удалось записать в файл %s: %v", filename, err)
	}
	return nil
}

func isGoFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".go")
}

func determineFileTypeFromContent(content string) bool {
	goIndicators := []string{"func ", "package ", "import (", "type ", "var ", "go func"}
	tsIndicators := []string{"function ", "const ", "export ", "interface ", "import ", "class ", "=>", "async function", "public ", "private ", ": void", ": string", ": number", ": boolean", "<T>"}

	goScore := 0
	tsScore := 0

	normalizedContent := strings.ToLower(content)

	for _, indicator := range goIndicators {
		if strings.Contains(normalizedContent, indicator) {
			goScore++
		}
	}
	for _, indicator := range tsIndicators {
		if strings.Contains(normalizedContent, indicator) {
			tsScore++
		}
	}

	if goScore == 0 && tsScore == 0 {
		return true
	}

	return goScore > tsScore
}

func parseArgs() (sourceFile string, targetFile string, useClipboard bool, valid bool) {
	args := os.Args[1:]

	if len(args) == 0 {
		return "", "", false, false
	}

	if len(args) >= 1 && (args[0] == "--clipboard" || args[0] == "-c") {
		if len(args) == 2 {
			return "", args[1], true, true
		}
		return "", "", false, false
	}

	sepIndex := -1
	for i, arg := range args {
		if arg == "--" {
			sepIndex = i
			break
		}
	}

	if sepIndex != -1 {
		argsBeforeSep := args[:sepIndex]
		argsAfterSep := args[sepIndex+1:]

		if len(argsBeforeSep) > 0 {
			return "", "", false, false
		}

		switch len(argsAfterSep) {
		case 1:
			return "", argsAfterSep[0], true, true
		case 2:
			return argsAfterSep[0], argsAfterSep[1], false, true
		default:
			return "", "", false, false
		}
	} else {
		switch len(args) {
		case 1:
			return "", args[0], true, true
		case 2:
			return args[0], args[1], false, true
		default:
			return "", "", false, false
		}
	}
}

func showUsage() {
	cmd := filepath.Base(os.Args[0])
	fmt.Printf("Использование:\n")
	fmt.Printf("  %s <целевой_файл>                             # Исходник из буфера обмена\n", cmd)
	fmt.Printf("  %s --clipboard <целевой_файл>                 # Исходник из буфера обмена (явно)\n", cmd)
	fmt.Printf("  %s -c <целевой_файл>                          # Исходник из буфера обмена (явно, коротко)\n", cmd)
	fmt.Printf("  %s <исходный_файл> <целевой_файл>             # Из файла в файл\n", cmd)
	fmt.Printf("  %s -- <целевой_файл>                          # Исходник из буфера обмена (с разделителем)\n", cmd)
	fmt.Printf("  %s -- <исходный_файл> <целевой_файл>          # Из файла в файл (с разделителем)\n", cmd)
	fmt.Println("\nПримеры:")
	fmt.Printf("  %s target.go\n", cmd)
	fmt.Printf("  %s --clipboard target.go\n", cmd)
	fmt.Printf("  %s source.go target.go\n", cmd)
	fmt.Printf("  %s -- source.go target.go\n", cmd)
	fmt.Println("\nПри использовании с go run (из каталога проекта):")
	fmt.Println("  go run . target.go")
	fmt.Println("  go run . -- source.go target.go")
}

func main() {
	sourceFile, targetFile, useClipboard, valid := parseArgs()
	if !valid {
		showUsage()
		os.Exit(1)
	}

	if useClipboard {
		log.Printf("Синхронизация функций из буфера обмена в %s\n", targetFile)
	} else {
		log.Printf("Синхронизация функций из %s в %s\n", sourceFile, targetFile)
	}

	replacer := NewFunctionReplacer()
	var sourceContent string
	var sourceIsGo bool
	var err error

	if useClipboard {
		sourceContent, err = readFromClipboard()
		if err != nil {
			log.Fatalf("Ошибка чтения из буфера обмена: %v", err)
		}
		sourceIsGo = determineFileTypeFromContent(sourceContent)
		log.Printf("Обнаружен тип исходного кода (из буфера): %s\n", map[bool]string{true: "Go", false: "TypeScript"}[sourceIsGo])
	} else {
		sourceContent, err = readFile(sourceFile)
		if err != nil {
			log.Fatalf("Ошибка чтения исходного файла '%s': %v", sourceFile, err)
		}
		sourceIsGo = isGoFile(sourceFile)
	}

	if _, statErr := os.Stat(targetFile); os.IsNotExist(statErr) {
		log.Printf("Целевой файл %s не существует. Он будет создан.", targetFile)
	}

	targetContentOriginal, err := readFile(targetFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalf("Ошибка чтения целевого файла '%s': %v", targetFile, err)
		}
		targetContentOriginal = ""
		log.Printf("Целевой файл %s не найден, будет создан новый.", targetFile)
	}

	targetIsGo := isGoFile(targetFile)

	if sourceIsGo != targetIsGo {
		sourceTypeStr := "TypeScript"
		if sourceIsGo {
			sourceTypeStr = "Go"
		}
		targetTypeStr := "TypeScript"
		if targetIsGo {
			targetTypeStr = "Go"
		}
		log.Fatalf("Типы исходного (%s) и целевого (%s) файлов не совпадают. Оба должны быть Go или оба TypeScript.", sourceTypeStr, targetTypeStr)
	}

	sourceFunctions, err := replacer.extractFunctions(sourceContent, sourceIsGo)
	if err != nil {
		log.Fatalf("Ошибка извлечения функций из исходного кода: %v", err)
	}
	log.Printf("Найдено %d функций в исходном коде.\n", len(sourceFunctions))

	updatedContent := replacer.replaceFunctions(targetContentOriginal, sourceFunctions, targetIsGo)

	if err := writeFile(targetFile, updatedContent); err != nil {
		log.Fatalf("Ошибка записи в целевой файл '%s': %v", targetFile, err)
	}

	log.Printf("Синхронизация завершена успешно для %s.\n", targetFile)
}
