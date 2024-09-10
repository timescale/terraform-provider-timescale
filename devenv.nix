{ pkgs, lib, config, inputs, ... }: {
  packages = [ pkgs.git ];

  languages = {
    go.enable = true;
    terraform.enable = true;
  };

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

  outputs = let
    name = "terraform-provider-timescale";
    version = "1.11.0";
    app = import ./app.nix { inherit pkgs name version; };
  in {
    app = app;
    image = import ./image.nix { inherit pkgs app name version; };
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
