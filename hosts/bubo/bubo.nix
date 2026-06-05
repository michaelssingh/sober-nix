{ pkgs, ... }:

pkgs.dockerTools.buildLayeredImage {
  name = "sober-bubo";
  tag = "latest";

  contents = [
    pkgs.forgejo
    pkgs.bashInteractive
    pkgs.coreutils
    pkgs.git
  ];

  config = {
    Entrypoint = [ "${pkgs.forgejo}/bin/forgejo" "web" ];
    ExposedPorts = {
      "3000/tcp" = { };
      "2222/tcp" = { };
    };
    Env = [
      "PATH=/bin"
    ];
  };
}
