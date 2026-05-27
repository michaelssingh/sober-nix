{ pkgs, ... }:

let
  # Basic entrypoint to run the nix-daemon
  entrypoint = pkgs.writeShellScriptBin "entrypoint" ''
    set -e
    # Run the nix-daemon. It needs to listen on the specified port.
    # We might need to ensure /nix/var/nix/daemon-socket exists if it uses a socket,
    # but for TCP, we just need to make sure the daemon is started.
    exec ${pkgs.nix}/bin/nix-daemon
  '';
in
pkgs.dockerTools.buildLayeredImage {
  name = "sober-styx";
  tag = "latest";

  contents = [
    pkgs.nix
    pkgs.bash
    pkgs.coreutils
    entrypoint
  ];

  config = {
    Entrypoint = [ "${entrypoint}/bin/entrypoint" ];
    ExposedPorts = {
      "3000/tcp" = { };
    };
    # Set the port for the nix-daemon
    Env = [ "NIX_DAEMON_PORT=3000" "PATH=/bin" ];
  };
}
