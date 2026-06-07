{ pkgs, config, ... }:

let
  # Dynamically inject the theme style into the Lua config
  luaConfig = builtins.replaceStrings 
    [ "style = \"storm\"" ] 
    [ "style = \"${config.sober.theme.current.variant}\"" ] 
    (builtins.readFile ./config.lua);
in
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
      # LSPs
      gopls
      rust-analyzer
      clang-tools
      lua-language-server
      nixd
      nixfmt-rfc-style
      harper
    ];
    plugins = with pkgs.vimPlugins; [
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
      # UI
      telescope-nvim
      telescope-fzf-native-nvim
      tokyonight-nvim
      which-key-nvim
      gitsigns-nvim
      lualine-nvim
      indent-blankline-nvim
      oil-nvim
      toggleterm-nvim
      # Treesitter (Optimized grammar list)
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
    ];
    withRuby = false;
    withPython3 = false;
    initLua = luaConfig;
  };
}
