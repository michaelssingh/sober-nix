{ lib, buildGoModule }:

buildGoModule {
  pname = "tyto";
  version = "0.1.0";
  src = ./.;
  vendorHash = "sha256-mXkZyunZzqg0wR181ym9nm7T2uBNKlKOOZeAMQBjvV0=";
}
