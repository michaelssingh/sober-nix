{ pkgs, inputs, ... }:
{
  programs.neovim = {
    enable = true;
    defaultEditor = true;
    viAlias = true;
    vimAlias = true;
    extraPackages = with pkgs; [
      ripgrep
      fd
      tree-sitter
      nodejs
    ];
    plugins = with pkgs.vimPlugins; [
      (pkgs.vimUtils.buildVimPlugin {
        name = "gemini-nvim";
        src = pkgs.fetchFromGitHub {
          owner = "kiddos";
          repo = "gemini.nvim";
          rev = "master";
          sha256 = "0gr2wyblr9vzxkwl045mz7sc8v36l10909bpmrlsz1ym0h9qskb7";
        };
      })
      nvim-lspconfig
      nvim-autopairs
      nvim-web-devicons
      # Autocompletion
      nvim-cmp
      cmp-nvim-lsp
      cmp-buffer
      cmp-path
      cmp_luasnip
      luasnip
      (nvim-treesitter.withPlugins (p: [
        p.nix
        p.lua
        p.vim
        p.bash
        p.markdown
        p.c
        p.go
        p.rust
      ]))
      nvim-treesitter-textobjects
      tokyonight-nvim
      telescope-nvim
      telescope-fzf-native-nvim
      which-key-nvim
      gitsigns-nvim
      lualine-nvim
      indent-blankline-nvim
      oil-nvim
      toggleterm-nvim
    ];
    withRuby = false;
    withPython3 = false;
    initLua = builtins.readFile ./config.lua;
  };
}
