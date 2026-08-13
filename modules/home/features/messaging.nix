{ pkgs, ... }:

{
  home.packages = with pkgs; [
    pkgs.senpai
    pkgs.aerc
    pkgs.neomutt
    pkgs.nheko
  ];
}
