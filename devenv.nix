{ pkgs, lib, config, inputs, ... }: {
  packages = [ pkgs.git pkgs.gomod2nix ];

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
  };

  outputs = let
    name = "terraform-provider-timescale";
    version = "1.11.0";
    app = import ./default.nix { inherit pkgs name version; };
  in {
    app = app;
    image = import ./image.nix { inherit pkgs app name version; };
  };

  tasks = {
    "dev:mod" = {
      exec = ''
        go mod download
        gomod2nix
      '';
      before = [ "dev:gen" "dev:diff" ];
    };

    "dev:gen" = {
      exec = ''
        go generate ./...
      '';
    };

    "dev:diff" = {
      exec = ''
        git diff --compact-summary --exit-code || \
         (echo; echo "Unexpected difference in directories after code generation. Run task 'dev:gen' and commit."; exit 1)
      '';
    };
  };
}
