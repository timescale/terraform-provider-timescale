{ pkgs, lib, config, inputs, ... }:
let
  providerBin = pkgs.buildGoModule {
    pname = "terraform-provider-timescale";
    version = "1.11.0";
    src = ./.;
    vendorHash = "sha256-ZDVMtvb49psIN+F4tABKl03HUvx/h6aOPs0Oni+KqqQ=";
  };
in {
  packages = [ pkgs.git ];

  languages.go.enable = true;
  languages.terraform.enable = true;

  pre-commit.hooks = {
    govet = {
      enable = true;
      pass_filenames = false;
    };
    gotest.enable = true;
    golangci-lint = {
      enable = true;
      pass_filenames = false;
    };
    generate-check = {
      enable = true;
      name = "Go generate checks";
      entry = ''
        run-diff
      '';
      pass_filenames = false;
    };
  };

  scripts = {
    run-mod-download.exec = ''
      go mod download
    '';
    run-generate.exec = ''
      go generate ./...
    '';
    run-diff.exec = ''
      git diff --compact-summary --exit-code || \
        (echo; echo "Unexpected difference in directories after code generation. Run 'run-generate' command and commit."; exit 1)
    '';
  };
}
