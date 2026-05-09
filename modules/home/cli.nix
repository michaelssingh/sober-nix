{ pkgs, ... }:

{

  # --- Terminal Multiplexer (Zellij) ---
  programs.zellij = {
    enable = true;
    settings = {
      theme = "tokyonight";
      default_mode = "locked";
      pane_frames = false;
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

    aichat
    gemini-cli
  ];

  # Configuration for aichat
  xdg.configFile."aichat/config.yaml".text = ''
    model: gemini:gemini-2.5-flash
    clients:
      - type: gemini
        api_key: "***REDACTED***"
    # Define your persona here
    roles:
      - name: assistant
        prompt: "Act as a Senior Systems Engineering Research Assistant. Zero Fluff, Technical Precision, Recursive Definition, Logical Rationale, Verifiable Sourcing."
  '';
}
