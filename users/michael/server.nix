{ pkgs, ... }:

{
  imports = [
    ./core.nix # <--- INHERITS THE BASE
  ];

  # Server specific tools?
  # home.packages = [ pkgs.docker-compose ];
}
