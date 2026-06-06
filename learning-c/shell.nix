{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  name = "c-learning-env";
  buildInputs = with pkgs; [
    gcc
    gdb
    valgrind
    gnumake
    clang-tools # Includes clang-format and clangd language server support
  ];

  shellHook = ''
    echo "================================================="
    echo "  Welcome to your C programming sandbox!         "
    echo "  Tools loaded: gcc, gdb, valgrind, make         "
    echo "  Run 'make' to compile, or './hello' to execute."
    echo "================================================="
  '';
}
