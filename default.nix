{ pkgs, name, version, ... }:
pkgs.buildGoModule {
  pname = name;
  version = version;
  src = ./.;
  vendorHash = "sha256-ZDVMtvb49psIN+F4tABKl03HUvx/h6aOPs0Oni+KqqQ=";
}
