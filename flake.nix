{
  description = "Function replacer for Go and TypeScript files";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = {
          default = self.packages.${system}.replacer;

          replacer = pkgs.buildGoModule {
            pname = "replacer";
            version = "0.1.0";

            src = ./.;

            vendorHash = "sha256-ewCKket3ARSY+AQLjWRdauEl5fMdamNXWCk3WMRjgBk=";

            meta = with pkgs.lib; {
              description = "Синхронизация функций между Go и TypeScript файлами";
              homepage = "https://github.com/username/replacer";
              license = licenses.mit;
              maintainers = [ "username" ];
            };
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
            delve
            nodejs
            typescript
          ];

          shellHook = ''
            echo "🔧 Окружение разработки replacer готово!"
            echo "Go version: $(go version)"
            echo "Node version: $(node --version)"
            echo ""
            echo "Доступные команды:"
            echo "  go run . target.go           # Исходник из буфера → target.go"
            echo "  go run . source.go target.go # source.go → target.go"
            echo "  go test ./...                # Запуск тестов"
            echo "  go build                     # Сборка"
          '';
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.replacer}/bin/replacer";
        };
      }
    );
}
