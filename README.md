# Replacer

Утилита для синхронизации функций между файлами Go и TypeScript.

## Возможности

- 🔄 Синхронизация функций между файлами
- 📋 Работа с буфером обмена
- 🐹 Поддержка Go (включая методы с receiver'ами)
- 📜 Поддержка TypeScript
- 🛠️ Простой интерфейс командной строки

## Установка

### С помощью Nix Flakes (рекомендуется)

```bash
# Разовый запуск
nix run github:username/replacer -- target.go

# Установка в профиль
nix profile install github:username/replacer

# Окружение разработки
nix develop
```

### Традиционный Nix

```bash
nix-build
./result/bin/replacer target.go
```

### Go

```bash
go install github.com/username/replacer@latest
```

## Использование

```bash
# Исходник из буфера обмена → target.go
replacer target.go

# source.go → target.go
replacer source.go target.go

# С явным разделителем
replacer -- target.go
replacer -- source.go target.go
```

## Разработка

```bash
# Запуск окружения разработки (Nix)
nix develop

# Сборка
make build

# Тесты
make test

# Пример
make run-example

# Полная проверка
make check
```

## Лицензия

MIT
```

## .gitignore
```
# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
go.work

# Build
/build/
/dist/

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Nix
result
result-*

# Temporary files
*.tmp
*.temp
```

Теперь у тебя есть полноценный репозиторий с современной инфраструктурой Nix! Основные изменения и дополнения:

1. **Nix Flakes** - современное управление зависимостями
2. **Тесты** - покрывают основную функциональность
3. **Makefile** - упрощает разработку
4. **go.mod** - управление Go зависимостями
5. **Документация** - подробное README

Для начала работы:
```bash
# Окружение разработки
nix develop

# Тесты
make test

# Сборка
make build
