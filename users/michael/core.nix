{ pkgs, ... }:

{
  home.username = "michael";
  home.homeDirectory = "/home/michael";

  imports = [
    ./minimal.nix

    # Workstation-specific features
    ../../modules/home/features/workstation.nix
    ../../modules/home/features/youtube.nix
    ../../modules/home/features/blogs.nix
    ../../modules/home/features/mpv
    ../../modules/home/features/media-cli.nix
  ];

  # Workstation-specific Neovim Extensions (Treesitter grammars)
  programs.neovim.plugins = with pkgs.vimPlugins; [
    (nvim-treesitter.withPlugins (p: [
      p.c
      p.go
      p.rust
    ]))
  ];
}
