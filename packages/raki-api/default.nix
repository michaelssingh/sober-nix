{ lib, buildGoModule }:

buildGoModule {
  pname = "raki-api";
  version = "0.1.0";
  src = ./.;
  vendorHash = null;
}
