{ pkgs, inputs, ... }:

{
  home.packages = with pkgs; [
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
    (pkgs.writeShellScriptBin "ani-cli" ''
      ${pkgs.bash}/bin/bash ${inputs.ani-cli}/ani-cli "$@"
    '')
    pkgs.socat
    pkgs.w3m
    pkgs.chafa
    pkgs.ani-skip
  ];

  home.file.".w3m/config".text = ''
    ext_image_viewer 1
    external_image_viewer ${pkgs.chafa}/bin/chafa
    extbrowser ${pkgs.firefox}/bin/firefox %s
  '';
}
