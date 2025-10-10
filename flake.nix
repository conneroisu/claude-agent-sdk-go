{
  description = "A development shell for go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    treefmt-nix.url = "github:numtide/treefmt-nix";
    treefmt-nix.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = {
    nixpkgs,
    flake-utils,
    treefmt-nix,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {
        inherit system;
        overlays = [
          (final: prev: {
            # Add your overlays here
            # Example:
            # my-overlay = final: prev: {
            #   my-package = prev.callPackage ./my-package { };
            # };
            final.buildGoModule = prev.buildGo125Module;
            buildGoModule = prev.buildGo125Module;
          })
        ];
      };

      rooted = exec:
        builtins.concatStringsSep "\n"
        [
          ''REPO_ROOT="$(git rev-parse --show-toplevel)"''
          exec
        ];

      scripts = {
        dx = {
          exec = rooted ''$EDITOR "$REPO_ROOT"/flake.nix'';
          description = "Edit flake.nix";
        };
        lint = {
          exec = rooted ''
            cd "$REPO_ROOT"
            golangci-lint run
            cd -
          '';
          description = "Lint";
        };
        tests = {
          exec = rooted ''
            cd "$REPO_ROOT"
            go test -v ./...
            cd -
          '';
          description = "Run tests";
        };
      };

      scriptPackages =
        pkgs.lib.mapAttrs
        (
          name: script:
            pkgs.writeShellApplication {
              inherit name;
              text = script.exec;
              runtimeInputs = script.deps or [];
            }
        )
        scripts;

      treefmtModule = {
        projectRootFile = "flake.nix";
        programs = {
          alejandra.enable = true;
          gofmt.enable = true;
          golines.enable = true;
          goimports.enable = true;
        };
      };
    in {
      devShells.default = pkgs.mkShell {
        name = "dev";

        packages = with pkgs;
          [
            alejandra # Nix
            nixd
            statix
            deadnix

            go_1_25 # Go Tools
            air
            golangci-lint
            gopls
            revive
            golines
            golangci-lint-langserver
            gomarkdoc
            gotests
            gotools
            reftools
            pprof
            graphviz
            goreleaser
            mdformat
          ]
          ++ builtins.attrValues scriptPackages;
      };

      formatter = treefmt-nix.lib.mkWrapper pkgs treefmtModule;
    });
}
