{ pkgs, ... }:

{
  home.packages = with pkgs; [
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
