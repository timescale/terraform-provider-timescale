{ pkgs, lib, config, inputs, ... }: {
  packages = with pkgs; [ git terraform golangci-lint ];

  languages.go.enable = true;

  scripts = {
    run_ci_lint.exec = ''
      golangci-lint run
    '';
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

  enterTest = ''
    go test -v -cover ./internal/provider/
  '';
}
