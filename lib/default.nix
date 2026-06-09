{ pkgs }:

{
  mkContainerImage =
    {
      name,
      tag ? "latest",
      packages ? [ ],
      extraContents ? [ ],
      entrypoint,
      exposedPorts ? { },
      env ? { },
      users ? { },
      groups ? { },
      harden ? true,
      usrBinEnv ? false,
    }:
    let
      # Base shell - use minimal non-interactive bash for hardened builds
      shellPkg = if harden then pkgs.bash else pkgs.bashInteractive;

      # Dynamically resolve the binary name of the shell
      shellBinName =
        if pkgs.lib.hasPrefix "bash" (shellPkg.pname or "") then "bash" else (shellPkg.pname or "sh");
      shellPath = "${shellPkg}/bin/${shellBinName}";

      # Core system tools always needed by the container runtime
      baseContents = [
        shellPkg
        pkgs.cacert
      ]
      ++ (if harden then [ ] else [ pkgs.coreutils ]);

      # Write files avoiding formatting line-break issues
      passwdContent = ''
        root:x:0:0::/root:${shellPath}
        ${pkgs.lib.concatStrings (
          pkgs.lib.mapAttrsToList (
            uName: u:
            "${uName}:x:${toString u.uid}:${toString u.gid}:${u.description or uName}:${u.home or "/var/empty"}:${u.shell or shellPath}\n"
          ) users
        )}'';

      userGroups = pkgs.lib.mapAttrs' (
        username: u: pkgs.lib.nameValuePair username { inherit (u) gid; }
      ) users;

      allGroups = userGroups // groups;

      groupContent = ''
        root:x:0:
        ${pkgs.lib.concatStrings (
          pkgs.lib.mapAttrsToList (gName: g: "${gName}:x:${toString g.gid}:\n") allGroups
        )}'';

      passwdDir = pkgs.writeTextDir "etc/passwd" passwdContent;
      groupDir = pkgs.writeTextDir "etc/group" groupContent;

      # Environment layout components
      envDerivations = [
        passwdDir
        groupDir
      ]
      ++ (pkgs.lib.optional usrBinEnv (
        pkgs.runCommand "usr-bin-env" { } ''
          mkdir -p $out/usr/bin
          ln -s ${if harden then "${pkgs.busybox}/bin/env" else "${pkgs.coreutils}/bin/env"} $out/usr/bin/env
        ''
      ))
      ++ [
        (pkgs.runCommand "tmp-dir" { } ''
          mkdir -p $out/tmp
          chmod 1777 $out/tmp
        '')
      ];

      # Consolidate configuration files into one single layer to prevent layer limit exceptions
      envLayer = pkgs.symlinkJoin {
        name = "${name}-env-layer";
        paths = envDerivations;
      };

      # Setup wrapper script
      entrypointScript = pkgs.writeShellScriptBin "entrypoint" ''
        set -e
        ${entrypoint}
      '';

      standardEnv = {
        PATH = pkgs.lib.makeBinPath (
          baseContents
          ++ packages
          ++ [ entrypointScript ]
          ++ (if usrBinEnv && harden then [ pkgs.busybox ] else [ ])
        );
        SSL_CERT_FILE = "${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt";
      };

      finalEnv = pkgs.lib.mapAttrsToList (k: v: "${k}=${v}") (standardEnv // env);
    in
    pkgs.dockerTools.buildLayeredImage {
      inherit name tag;

      contents =
        baseContents
        ++ [
          envLayer
          entrypointScript
        ]
        ++ packages
        ++ extraContents;

      config = {
        Entrypoint = [ "${entrypointScript}/bin/entrypoint" ];
        ExposedPorts = exposedPorts;
        Env = finalEnv;
      };
    };
}
