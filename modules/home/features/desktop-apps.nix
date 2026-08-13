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
    terminus_font
    spleen
    qutebrowser
    antigravity
    strix-paste
    imv
  ];

  # Terminal Multiplexer
  programs.zellij = {
    enable = true;
    settings = {
      theme = "tokyonight";
      default_mode = "locked";
      pane_frames = false;
      mouse_mode = true;
      copy_on_select = true;
      mirror_session = true;

      themes.tokyonight = {
        fg = "#c0caf5";
        bg = "#1a1b26";
        black = "#15161e";
        red = "#f7768e";
        green = "#9ece6a";
        yellow = "#e0af68";
        blue = "#7aa2f7";
        magenta = "#bb9af7";
        cyan = "#7dcfff";
        white = "#a9b1d6";
        orange = "#ff9e64";
      };

      ui = {
        pane_frames = {
          rounded_corners = true;
        };
      };
    };
  };
}
