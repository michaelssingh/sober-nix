{ pkgs }:

pkgs.buildGoModule {
  pname = "appservice-mgr";
  version = "0.1.5";
  src = ./.;

  vendorHash = null; # Use null for now as we have no external dependencies other than standard lib (wait, we have yaml.v3)
}
