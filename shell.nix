{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    gopls
    gotools
    go-tools
    delve
    nodejs
    typescript

    # Дополнительные утилиты для разработки
    git
    gnumake

    # Linux специфичные зависимости для clipboard
    xclip
    wl-clipboard
  ];

  shellHook = ''
    echo "🔧 Окружение разработки replacer готово!"
    echo "Go version: $(go version)"
    echo ""
    echo "Доступные команды:"
    echo "  go run . target.go           # Исходник из буфера → target.go"
    echo "  go run . source.go target.go # source.go → target.go"
    echo "  go test ./...                # Запуск тестов"
    echo "  go build                     # Сборка"
    echo ""
    echo "Для работы с буфером обмена в Linux может понадобиться xclip или wl-clipboard"
  '';
}
