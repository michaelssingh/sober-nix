{
  pkgs,
  inputs,
  ...
}:

{
  home.packages = with pkgs; [
    foot
    fuzzel
    swaylock
    swayidle
    inputs.nixpkgs-pinned.legacyPackages.${pkgs.stdenv.hostPlatform.system}.transmission_4
    stig
    chawan
    qutebrowser
    antigravity
    strix-paste
    imv
  ];
}
