{ pkgs, inputs, ... }:
{
  programs.neovim = {
    enable = true;
    defaultEditor = true;
    viAlias = true;
    vimAlias = true;
    extraPackages = with pkgs; [
      nixd
      lua-language-server
      ripgrep
      fd
    ];
    plugins = with pkgs.vimPlugins; [
      (pkgs.vimUtils.buildVimPlugin {
        name = "gemini-cli-nvim";
        src = inputs.gemini-nvim;
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
    extraLuaConfig = builtins.readFile ./config.lua;
  };
}
