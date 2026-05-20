{ pkgs, ... }:

{
  imports = [ ./dev.nix ];

  # --- Terminal Multiplexer (Zellij) ---
  programs.zellij = {
    enable = true;
    settings = {
      theme = "tokyonight";
      default_mode = "locked"; # Start in locked mode to avoid key collisions
      pane_frames = false;
      mouse_mode = true; # Easier to learn with mouse support enabled
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

      # Help beginners by showing the keybind bar until learned
      ui = {
        pane_frames = {
          rounded_corners = true;
        };
      };
    };
  };
  home.packages = with pkgs; [
    # DevOps / Cloud
    k9s
    kubectl
    opentofu
    awscli2
    oci-cli

    # Utilities
    jq
    yq-go
    htop
    btop
    nh # Nix Helper

    gemini-cli
    antigravity
    himalaya
    flyctl
    hydroxide

    pkgs.socat
    pkgs.senpai
    pkgs.harper
    ];
}
