{
  description = "immich-go development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go toolchain
            go_1_25
            gopls
            delve

            # Linting & security
            golangci-lint
            govulncheck

            # Release tooling
            goreleaser

            # Useful extras
            git
          ];

          shellHook = ''
            echo "immich-go dev shell ready — Go $(go version | cut -d' ' -f3)"
          '';
        };
      });
}
