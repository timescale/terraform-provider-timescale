{ pkgs, lib, config, inputs, ... }:
let
  providerBin = pkgs.buildGoModule {
    pname = "terraform-provider-timescale";
    version = "1.11.0";
    src = ./.;
    vendorHash = "sha256-ZDVMtvb49psIN+F4tABKl03HUvx/h6aOPs0Oni+KqqQ=";
  };
in {
  packages = with pkgs; [ git terraform ];

  languages.go.enable = true;

  pre-commit.src = ./.;
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
  };

  scripts = {
    run_mod_download.exec = ''
      go mod download
    '';
    run_build.exec = ''
      go build -v .
    '';
    run_generate.exec = ''
      go generate ./...
    '';
    run_diff.exec = ''
      git diff --compact-summary --exit-code || \
        (echo; echo "Unexpected difference in directories after code generation. Run 'go generate ./...' command and commit."; exit 1)
    '';
  };
}
