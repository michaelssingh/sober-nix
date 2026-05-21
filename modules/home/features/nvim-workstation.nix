{ pkgs, ... }:

{
  programs.neovim.plugins = with pkgs.vimPlugins; [
    (nvim-treesitter.withPlugins (p: [
      p.c
      p.go
      p.rust
    ]))
  ];
}
