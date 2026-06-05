{ pkgs }:

{
  mkContainerImage = {
    name,
    tag ? "latest",
    packages ? [],
    extraContents ? [],
    entrypoint,
    exposedPorts ? {},
    env ? {},
    users ? {},
    harden ? true,
    usrBinEnv ? false,
  }:
  let
    # Base shell - use minimal non-interactive bash for hardened builds
    shellPkg = if harden then pkgs.bash else pkgs.bashInteractive;

    # Core system tools always needed by the container runtime
    baseContents = [
      shellPkg
      pkgs.cacert
    ] ++ (if harden then [] else [ pkgs.coreutils ]);

    # Write passwd and group files dynamically
    passwdContent = ''
      root:x:0:0::/root:${shellPkg}/bin/bash
    '' + (pkgs.lib.concatStringsSep "\n" (pkgs.lib.mapAttrsToList (username: u:
      "${username}:x:${toString u.uid}:${toString u.gid}:${u.description or username}:${u.home or "/var/empty"}:${u.shell or "${shellPkg}/bin/bash"}"
    ) users)) + "\n";

    groupContent = ''
      root:x:0:
    '' + (pkgs.lib.concatStringsSep "\n" (pkgs.lib.mapAttrsToList (groupname: g:
      "${groupname}:x:${toString g.gid}:"
    ) users)) + "\n";

    passwdDir = pkgs.writeTextDir "etc/passwd" passwdContent;
    groupDir = pkgs.writeTextDir "etc/group" groupContent;

    # Environment layout setup
    envDerivations = [
      passwdDir
      groupDir
    ] ++ (pkgs.lib.optional usrBinEnv (pkgs.runCommand "usr-bin-env" {} ''
      mkdir -p $out/usr/bin
      ln -s ${pkgs.coreutils}/bin/env $out/usr/bin/env
    '')) ++ [
      # Standard /tmp directory with sticky-bit permissions (required by standard application runtimes)
      (pkgs.runCommand "tmp-dir" {} ''
        mkdir -p $out/tmp
        chmod 1777 $out/tmp
      '')
    ];

    # Setup wrapper script
    entrypointScript = pkgs.writeShellScriptBin "entrypoint" ''
      set -e
      # Execute the custom application startup commands
      ${entrypoint}
    '';

    standardEnv = {
      PATH = "/bin";
      SSL_CERT_FILE = "${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt";
    };

    finalEnv = pkgs.lib.mapAttrsToList (k: v: "${k}=${v}") (standardEnv // env);
  in
  pkgs.dockerTools.buildLayeredImage {
    inherit name tag;

    contents = baseContents ++ envDerivations ++ [ entrypointScript ] ++ packages ++ extraContents;

    config = {
      Entrypoint = [ "${entrypointScript}/bin/entrypoint" ];
      ExposedPorts = exposedPorts;
      Env = finalEnv;
    };
  };
}
