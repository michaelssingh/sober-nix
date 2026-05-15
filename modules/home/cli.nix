{ pkgs, inputs, ... }:

{

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
    himalaya
    flyctl
    hydroxide

    (pkgs.writeShellScriptBin "mpv-queue" ''
      SOCKET="''${XDG_RUNTIME_DIR:-/tmp}/mpv-socket"
      
      if [ -S "$SOCKET" ] && ${pkgs.socat}/bin/socat - "$SOCKET" <<< '{ "command": ["get_property", "idle-active"] }' >/dev/null 2>&1; then
        : # Socket exists and is responsive
      else
        rm -f "$SOCKET"
        # Use nohup and redirect streams to fully detach from the parent process group
        nohup ${pkgs.mpv}/bin/mpv --idle=yes --input-ipc-server="$SOCKET" --force-window=yes --really-quiet >/dev/null 2>&1 </dev/null &
        
        for i in {1..20}; do
          if [ -S "$SOCKET" ]; then break; fi
          sleep 0.1
        done
      fi
      
      echo '{ "command": ["loadfile", "'"$1"'", "append-play"] }' | ${pkgs.socat}/bin/socat - "$SOCKET"
    '')
    pkgs.socat
    pkgs.w3m
    pkgs.chafa
  ];

  home.file.".w3m/config".text = ''
    ext_image_viewer 1
    external_image_viewer ${pkgs.chafa}/bin/chafa
    extbrowser ${pkgs.firefox}/bin/firefox %s
  '';


  # Configuration for aichat
  xdg.configFile."aichat/config.yaml".text = ''
    model: gemini:gemini-2.5-flash
    clients:
      - type: gemini
        api_key: "AIzaSyAqX9sfLvhMhVXSrWi1zRhvrmQ2BumhMmg"
    # Define your persona here
    roles:
      - name: assistant
        prompt: "Act as a Senior Systems Engineering Research Assistant. Zero Fluff, Technical Precision, Recursive Definition, Logical Rationale, Verifiable Sourcing."
  '';
}
