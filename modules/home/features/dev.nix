{ pkgs, ... }:

{
  home.packages = with pkgs; [
    # LSP & Dev Tools
    nixd
    lua-language-server
    harper

    # C
    gcc
    gdb
    gnumake
    clang-tools

    # Go
    go
    gopls
    golangci-lint

    # Rust
    rustc
    cargo
    rust-analyzer
  ];
}
