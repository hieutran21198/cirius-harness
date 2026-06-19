{
  pkgs,
  lib,
  ...
}:
{
  # ---------------------------------------------------------------------------
  # Workspace dev environment
  #
  # Single entrypoint for everyone working in this monorepo. Run `direnv allow`
  # once, then every shell you open here gets the same toolchain. Avoids the
  # `it works on my machine` class of bug entirely.
  # ---------------------------------------------------------------------------

  env = {
    WORKSPACE_NAME = "cirius-harness";
    NIX_LD = "${pkgs.stdenv.cc.bintools.dynamicLinker}";
    NIX_LD_LIBRARY_PATH = lib.makeLibraryPath (
      with pkgs;
      [
        stdenv.cc.cc
        zlib
        openssl
        sqlite
        icu
        libuv
      ]
    );
  };

  # Tooling shared by every service / package / app. Keep this list small;
  # service-specific deps belong in that service's own devenv shell if needed.
  packages = with pkgs; [
    # secret scanning + commit hooks
    gitleaks
    pre-commit

    # go tooling (compiler is provided by languages.go below)
    golangci-lint
    gopls
    gotools

    # general
    jq
    yq-go

  ];

  languages = {
    go = {
      enable = true;
      version = "1.26.3";
    };
    javascript = {
      enable = true;
      package = pkgs.nodejs_22;
      # pnpm drives the root JS workspace + nx. Install handled by user (pnpm i),
      # not by devenv-up, so we don't fight CI caching.
      pnpm = {
        enable = true;
        package = pkgs.nodejs_22;
        install.enable = false;
      };
    };
  };

  # --------------------------------------------------------------------------
  # Git hooks (via cachix/git-hooks.nix - auto-installed on `devenv shell`)
  #   - Conventional Commits enforced on commit-msg
  #   - Secret / credential scanning on pre-commit
  #   - Basic repo hygiene
  # --------------------------------------------------------------------------
  git-hooks.hooks = {
    # Conventional Commits gate (commit-msg stage)
    commitizen.enable = true;

    # Secret / credential scanning (pre-commit stage)
    detect-aws-credentials.enable = true;
    detect-private-keys.enable = true;
    ripsecrets.enable = true;
    trufflehog.enable = true;

    # Repo hygiene
    check-added-large-files = {
      enable = true;
      # 1 MB ceiling - accommodates pnpm-lock.yaml & similar lockfiles.
      args = [ "--maxkb=1024" ];
    };
    check-merge-conflicts.enable = true;
    end-of-file-fixer.enable = true;
    trim-trailing-whitespace = {
      enable = true;
      # Preserve markdown line breaks (matches .editorconfig).
      args = [ "--markdown-linebreak-ext=md" ];
    };
    mixed-line-endings.enable = true;
  };

  # Tasks runnable via `devenv tasks run <name>`. Use these instead of a Makefile
  # so every contract is declared in one place that direnv already loads.
  tasks = {
    "workspace:bootstrap" = {
      description = "Install JS deps + sync Go workspace modules";
      exec = ''
        set -euo pipefail
        pnpm install --frozen-lockfile=false
        go work sync
      '';
    };

    "workspace:fmt" = {
      description = "Format Go + TS sources";
      exec = ''
        set -euo pipefail
        go work edit -fmt
        gofmt -w $(go list -f '{{.Dir}}' -m all 2>/dev/null | grep -v '^$' || true)
        pnpm -w exec nx format:write
      '';
    };

    "workspace:lint" = {
      description = "Lint Go modules + nx-managed TS libs";
      exec = ''
        set -euo pipefail
        for mod in $(find packages/go services -name go.mod -not -path '*/node_modules/*'); do
          dir=$(dirname "$mod")
          echo "==> golangci-lint in $dir"
          (cd "$dir" && golangci-lint run ./...)
        done
        pnpm -w exec nx run-many -t lint
      '';
    };

    "workspace:test" = {
      description = "Run Go module tests + nx-managed TS tests";
      exec = ''
        set -euo pipefail
        for mod in $(find packages/go services -name go.mod -not -path '*/node_modules/*'); do
          dir=$(dirname "$mod")
          echo "==> go test in $dir"
          (cd "$dir" && go test ./...)
        done
        pnpm -w exec nx run-many -t test
      '';
    };

    "db:migrate" = {
      description = "Apply harness DB migrations (embedded goose, pure-Go SQLite)";
      exec = ''
        set -euo pipefail
        go run ./services/harness/cmd/migrate up
      '';
    };

    "db:rollback" = {
      description = "Roll back the most recent harness DB migration";
      exec = ''
        set -euo pipefail
        go run ./services/harness/cmd/migrate down
      '';
    };

    "db:status" = {
      description = "Report applied/pending harness DB migrations";
      exec = ''
        set -euo pipefail
        go run ./services/harness/cmd/migrate status
      '';
    };
  };
}
