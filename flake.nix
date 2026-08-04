{
  description = "SyncWatch — live dashboard for ArgoCD auto-sync state";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    predictable-yaml = {
      url = "github:snarlysodboxer/predictable-yaml/v1.0.1";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, predictable-yaml }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: rec {
        syncwatch = pkgs.buildGoModule {
          pname = "syncwatch";
          version = "0.1.0";
          src = self;
          vendorHash = "sha256-yAm2VK+uW5MDACR4DxAiedyXKf15ZFsBVqUwymarrbQ=";
          env.CGO_ENABLED = 0;
          ldflags = [ "-s" "-w" ];
          meta.description = "Live dashboard for ArgoCD auto-sync state";
          meta.license = pkgs.lib.licenses.asl20;
          meta.mainProgram = "syncwatch";
        };
        default = syncwatch;
      });

      devShells = forAllSystems (pkgs:
        let
          predictable-yaml-pkg = predictable-yaml.packages.${pkgs.stdenv.hostPlatform.system}.default;

          pre-push = pkgs.writeShellApplication {
            name = "pre-push";
            runtimeInputs = [
              pkgs.gitMinimal
              predictable-yaml-pkg
            ];
            meta.description = "Pre-push checks: predictable-yaml";
            text = ''
              set -euo pipefail

              cd "$(git rev-parse --show-toplevel)"

              echo "Running pre-push checks..."

              echo "  predictable-yaml lint kustomize"
              predictable-yaml lint kustomize --quiet

              echo "All pre-push checks passed."
            '';
          };
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
              gopls
              gotools
              go-tools
              predictable-yaml-pkg
              pre-push
            ];

            shellHook = ''
              # Install pre-push git hook
              if [ -d .git/hooks ]; then
                ln -sf ${pre-push}/bin/pre-push .git/hooks/pre-push
              fi
            '';
          };
        });
    };
}
