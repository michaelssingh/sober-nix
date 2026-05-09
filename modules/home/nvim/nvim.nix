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
      (nvim-treesitter.withPlugins (p: [
        p.nix
        p.lua
        p.vim
        p.bash
        p.markdown
      ]))
      tokyonight-nvim
      telescope-nvim
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
