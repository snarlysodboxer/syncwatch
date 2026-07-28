{
  description = "SyncWatch — live dashboard for ArgoCD auto-sync state";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
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

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            go-tools
          ];
        };
      });
    };
}
