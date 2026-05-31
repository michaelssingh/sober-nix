{ pkgs, inputs, ... }:

{
  home.packages = with pkgs; [
    (pkgs.writeShellApplication {
      name = "mpv-queue";
      runtimeInputs = [
        socat
        mpv
        coreutils
      ];
      text = ''
        SOCKET="/tmp/mpv-socket"

        # 1. Identify the actual URL/File vs flags
        # We assume the last argument is the target if it doesn't look like a flag,
        # or we just take the first non-flag argument.
        TARGET=""
        args=("$@")
        for i in "''${!args[@]}"; do
          if [[ "''${args[$i]}" != -* ]]; then
            TARGET="''${args[$i]}"
            # Remove target from args so we only have flags left
            unset "args[$i]"
            break
          fi
        done

        if [ -S "$SOCKET" ] && socat - "$SOCKET" <<< '{ "command": ["get_property", "idle-active"] }' >/dev/null 2>&1; then
          # Already running: just append the target
          if [ -n "$TARGET" ]; then
            echo "{ \"command\": [\"loadfile\", \"$TARGET\", \"append-play\"] }" | socat - "$SOCKET"
          fi
        else
          # Not running: start fresh
          rm -f "$SOCKET"
          # Launch detached
          nohup mpv --idle=yes --input-ipc-server="$SOCKET" --force-window=yes --really-quiet "''${args[@]}" "$TARGET" >/dev/null 2>&1 </dev/null &
          
          # Wait for socket
          for _ in {1..20}; do
            if [ -S "$SOCKET" ]; then break; fi
            sleep 0.1
          done
        fi
      '';
    })
    pkgs.socat
    pkgs.w3m
    pkgs.chafa
    pkgs.ani-skip
    pkgs.tyto
    (pkgs.writeShellApplication {
      name = "ani-cli";
      runtimeInputs = [
        bash
        curl
        gnugrep
        gnused
        fzf
        yt-dlp
        ffmpeg
        openssl
        mpv
        ani-skip
        aria2
      ];
      text = ''
        bash "${inputs.ani-cli}/ani-cli" "$@"
      '';
    })
  ];

  home.file.".w3m/config".text = ''
    ext_image_viewer 1
    external_image_viewer ${pkgs.chafa}/bin/chafa
    extbrowser ${pkgs.firefox}/bin/firefox %s
  '';
}
