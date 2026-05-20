{ pkgs, ... }:

{
  imports = [
    ./minimal.nix # <--- INHERITS THE BASE
    ../../modules/home/features/soju.nix
  ];

  # Server specific tools?
  # home.packages = [ pkgs.docker-compose ];
}
