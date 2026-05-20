{ pkgs, ... }:

{
  imports = [
    ./minimal.nix

    # Workstation-specific features
    ../../modules/home/features/youtube.nix
    ../../modules/home/features/blogs.nix
    ../../modules/home/features/mpv
    ../../modules/home/features/media-cli.nix
  ];
}
