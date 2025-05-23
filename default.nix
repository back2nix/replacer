{ pkgs ? import <nixpkgs> {} }:

pkgs.buildGoModule {
  pname = "replacer";
  version = "0.1.0";

  src = ./.;

  vendorHash = "sha256-KQBSmB5dMBEw54adFnLu1pEKKKIBNgFEaL3ZcJPPMtM=";

  meta = with pkgs.lib; {
    description = "Синхронизация функций между Go и TypeScript файлами";
    homepage = "https://github.com/back2nix/replacer";
    license = licenses.mit;
    maintainers = [ "username" ];
  };
}
