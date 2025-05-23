package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/atotto/clipboard"
)

type Function struct {
	Name     string
	Receiver string
	FullText string
	StartPos int
	EndPos   int
}

type FunctionReplacer struct {
	goFuncRegex *regexp.Regexp
	tsFuncRegex *regexp.Regexp
}

func NewFunctionReplacer() *FunctionReplacer {
	// Regex для Go функций (с receiver и без)
	goRegex := regexp.MustCompile(`(?s)func\s*(?:\([^)]*\))?\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\([^)]*\)(?:\s*[^{]*)?{(?:[^{}]*{[^{}]*})*[^{}]*}`)

	// Regex для TypeScript функций
	tsRegex := regexp.MustCompile(`(?s)(?:export\s+)?(?:function\s+([a-zA-Z_][a-zA-Z0-9_]*)|const\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(?:\([^)]*\)\s*)?=>)\s*[^{]*{(?:[^{}]*{[^{}]*})*[^{}]*}`)

	return &FunctionReplacer{
		goFuncRegex: goRegex,
		tsFuncRegex: tsRegex,
	}
}

func (fr *FunctionReplacer) extractFunctions(content string, isGoFile bool) ([]Function, error) {
	var functions []Function

	if isGoFile {
		// Go функции и методы
		funcRegex := regexp.MustCompile(`(?s)func\s*(?:\([^)]*\))?\s*([A-Za-z_][A-Za-z0-9_]*)\s*\([^)]*\)\s*(?:[^{]*)?{(?:[^{}]*(?:{[^{}]*}[^{}]*)*)*}`)
		matches := funcRegex.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			if len(match) >= 2 {
				functions = append(functions, Function{
					Name:     match[1],
					Receiver: extractGoReceiver(match[0]),
					FullText: strings.TrimSpace(match[0]),
				})
			}
		}
	} else {
		// TypeScript функции и методы
		functions = append(functions, fr.extractTSFunctions(content)...)
		functions = append(functions, fr.extractTSClassMethods(content)...)
	}

	return functions, nil
}

func (fr *FunctionReplacer) extractTSFunctions(content string) []Function {
	var functions []Function

	// 1. Обычные функции: function name() {}
	funcRegex := regexp.MustCompile(`(?s)function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\([^)]*\)\s*{(?:[^{}]*(?:{[^{}]*}[^{}]*)*)*}`)
	matches := funcRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			functions = append(functions, Function{
				Name:     match[1],
				FullText: strings.TrimSpace(match[0]),
			})
		}
	}

	// 2. Export функции: export function name() {}
	exportFuncRegex := regexp.MustCompile(`(?s)export\s+function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\([^)]*\)\s*{(?:[^{}]*(?:{[^{}]*}[^{}]*)*)*}`)
	exportMatches := exportFuncRegex.FindAllStringSubmatch(content, -1)
	for _, match := range exportMatches {
		if len(match) >= 2 {
			functions = append(functions, Function{
				Name:     match[1],
				FullText: strings.TrimSpace(match[0]),
			})
		}
	}

	// 3. Стрелочные функции: const name = () => {}
	arrowFuncRegex := regexp.MustCompile(`(?s)(?:const|let|var)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\([^)]*\)\s*=>\s*{(?:[^{}]*(?:{[^{}]*}[^{}]*)*)*}`)
	arrowMatches := arrowFuncRegex.FindAllStringSubmatch(content, -1)
	for _, match := range arrowMatches {
		if len(match) >= 2 {
			functions = append(functions, Function{
				Name:     match[1],
				FullText: strings.TrimSpace(match[0]),
			})
		}
	}

	return functions
}

func (fr *FunctionReplacer) extractTSClassMethods(content string) []Function {
	var functions []Function

	// Ищем методы внутри классов: methodName() { ... }
	// Паттерн ищет методы, которые не являются конструкторами и не начинаются с ключевых слов
	methodRegex := regexp.MustCompile(`(?s)(?:\s{4,}|\t+)([A-Za-z_][A-Za-z0-9_]*)\s*\([^)]*\)\s*{(?:[^{}]*(?:{[^{}]*}[^{}]*)*)*}`)
	matches := methodRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			methodName := match[1]
			// Исключаем конструкторы и некоторые ключевые слова
			if methodName != "constructor" && methodName != "function" && methodName != "class" {
				functions = append(functions, Function{
					Name:     methodName,
					FullText: strings.TrimSpace(match[0]),
				})
			}
		}
	}

	return functions
}

func extractGoFunctionName(funcText string) string {
	// Более точное извлечение имени Go функции
	nameRegex := regexp.MustCompile(`func\s*(?:\([^)]*\))?\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
	matches := nameRegex.FindStringSubmatch(funcText)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func extractGoReceiver(funcText string) string {
	// Извлечение receiver для Go функций
	receiverRegex := regexp.MustCompile(`func\s*\(([^)]+)\)\s*[a-zA-Z_][a-zA-Z0-9_]*\s*\(`)
	matches := receiverRegex.FindStringSubmatch(funcText)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func (fr *FunctionReplacer) replaceFunctions(targetContent string, sourceFunctions []Function, isGoFile bool) string {
	result := targetContent

	// Извлекаем существующие функции из целевого файла
	targetFunctions, err := fr.extractFunctions(targetContent, isGoFile)
	if err != nil {
		log.Printf("Предупреждение: ошибка при парсинге целевого файла: %v", err)
		targetFunctions = []Function{} // Продолжаем с пустым списком
	}

	// Создаем карту существующих функций для быстрого поиска
	targetFuncMap := make(map[string]Function)
	for _, fn := range targetFunctions {
		key := fr.getFunctionKey(fn, isGoFile)
		targetFuncMap[key] = fn
	}

	// Обновляем существующие функции
	for _, sourceFn := range sourceFunctions {
		key := fr.getFunctionKey(sourceFn, isGoFile)
		if targetFn, exists := targetFuncMap[key]; exists {
			// Заменяем существующую функцию
			result = strings.Replace(result, targetFn.FullText, sourceFn.FullText, 1)
			delete(targetFuncMap, key) // Помечаем как обработанную
		}
	}

	// Добавляем новые функции в конец файла
	for _, sourceFn := range sourceFunctions {
		key := fr.getFunctionKey(sourceFn, isGoFile)
		if _, exists := targetFuncMap[key]; !exists {
			// Добавляем новую функцию
			result += "\n\n" + sourceFn.FullText
		}
	}

	return result
}

func (fr *FunctionReplacer) getFunctionKey(fn Function, isGoFile bool) string {
	if isGoFile && fn.Receiver != "" {
		// Для Go методов с receiver: "receiver.functionName"
		return fmt.Sprintf("%s.%s", fn.Receiver, fn.Name)
	}
	// Для обычных функций: просто имя
	return fn.Name
}

func readFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("не удалось открыть файл %s: %v", filename, err)
	}
	defer file.Close()

	var content strings.Builder
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		content.WriteString(scanner.Text() + "\n")
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("ошибка чтения файла %s: %v", filename, err)
	}

	return content.String(), nil
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
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("не удалось создать файл %s: %v", filename, err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("не удалось записать в файл %s: %v", filename, err)
	}

	return nil
}

func isGoFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".go")
}

func determineFileTypeFromContent(content string) bool {
	// Простая эвристика для определения типа файла по содержимому
	goIndicators := []string{"func ", "package ", "import (", "type ", "var "}
	tsIndicators := []string{"function ", "const ", "export ", "interface ", "import "}

	goScore := 0
	tsScore := 0

	for _, indicator := range goIndicators {
		if strings.Contains(content, indicator) {
			goScore++
		}
	}

	for _, indicator := range tsIndicators {
		if strings.Contains(content, indicator) {
			tsScore++
		}
	}

	return goScore > tsScore
}

func parseArgs() (sourceFile string, targetFile string, useClipboard bool, valid bool) {
	args := os.Args[1:]

	switch len(args) {
	case 1:
		// Один аргумент
		if args[0] == "--" {
			// Только разделитель без аргументов - ошибка
			return "", "", false, false
		}
		// program target (исходник из буфера обмена)
		return "", args[0], true, true

	case 2:
		if args[0] == "--" {
			// program -- target (исходник из буфера обмена, с разделителем)
			return "", args[1], true, true
		} else if args[0] == "--clipboard" || args[0] == "-c" {
			// program --clipboard target (исходник из буфера обмена, явно)
			return "", args[1], true, true
		}
		// program source target (из файла в файл)
		return args[0], args[1], false, true

	case 3:
		if args[0] == "--" {
			// program -- source target (из файла в файл, с разделителем)
			return args[1], args[2], false, true
		}
	}

	return "", "", false, false
}

func showUsage() {
	fmt.Printf("Использование:\n")
	fmt.Printf("  replacer <целевой_файл>                             # Исходник из буфера обмена\n")
	fmt.Printf("  replacer -- <целевой_файл>                          # Исходник из буфера обмена (с разделителем)\n")
	fmt.Printf("  replacer --clipboard <целевой_файл>                 # Исходник из буфера обмена (явно)\n")
	fmt.Printf("  replacer <исходный_файл> <целевой_файл>             # Из файла в файл\n")
	fmt.Printf("  replacer -- <исходный_файл> <целевой_файл>          # Из файла в файл (с разделителем)\n")
	fmt.Println("\nПримеры:")
	fmt.Println("  replacer target.go                                  # Код из буфера → target.go")
	fmt.Println("  replacer -- target.go                               # Код из буфера → target.go")
	fmt.Println("  replacer --clipboard target.go                      # Код из буфера → target.go")
	fmt.Println("  replacer source.go target.go                        # source.go → target.go")
	fmt.Println("  replacer -- source.go target.go                     # source.go → target.go")
	fmt.Println("\nПри использовании с go run:")
	fmt.Println("  go run replacer.go target.go")
	fmt.Println("  go run replacer.go -- target.go")
	fmt.Println("  go run replacer.go source.go target.go")
}

func main() {
	sourceFile, targetFile, useClipboard, valid := parseArgs()

	if !valid {
		showUsage()
		os.Exit(1)
	}

	if useClipboard {
		fmt.Printf("Синхронизация функций из буфера обмена в %s\n", targetFile)
	} else {
		fmt.Printf("Синхронизация функций из %s в %s\n", sourceFile, targetFile)
	}

	replacer := NewFunctionReplacer()

	// Читаем исходный контент
	var sourceContent string
	var sourceIsGo bool
	var err error

	if useClipboard {
		sourceContent, err = readFromClipboard()
		if err != nil {
			log.Fatalf("Ошибка чтения из буфера обмена: %v", err)
		}

		// Определяем тип по содержимому буфера обмена
		sourceIsGo = determineFileTypeFromContent(sourceContent)
		fmt.Printf("Обнаружен тип исходного кода: %s\n", map[bool]string{true: "Go", false: "TypeScript"}[sourceIsGo])
	} else {
		sourceContent, err = readFile(sourceFile)
		if err != nil {
			log.Fatalf("Ошибка чтения исходного файла: %v", err)
		}
		sourceIsGo = isGoFile(sourceFile)
	}

	// Читаем целевой файл
	targetContent, err := readFile(targetFile)
	if err != nil {
		log.Fatalf("Ошибка чтения целевого файла: %v", err)
	}

	// Определяем тип целевого файла
	targetIsGo := isGoFile(targetFile)

	if sourceIsGo != targetIsGo {
		log.Fatalf("Файлы должны быть одного типа (оба Go или оба TypeScript)")
	}

	// Извлекаем функции из исходного контента
	sourceFunctions, err := replacer.extractFunctions(sourceContent, sourceIsGo)
	if err != nil {
		log.Fatalf("Ошибка извлечения функций из исходного кода: %v", err)
	}

	fmt.Printf("Найдено %d функций в исходном коде\n", len(sourceFunctions))

	// Заменяем/добавляем функции в целевом файле
	updatedContent := replacer.replaceFunctions(targetContent, sourceFunctions, targetIsGo)

	// Записываем обновленное содержимое
	if err := writeFile(targetFile, updatedContent); err != nil {
		log.Fatalf("Ошибка записи в целевой файл: %v", err)
	}

	fmt.Printf("Синхронизация завершена успешно\n")
}
