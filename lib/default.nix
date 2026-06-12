{ pkgs }:
{
  mkContainerImage =
    {
      name,
      tag ? "latest",
      packages ? [ ],
      observability ? null,
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
      # Vector package
      vectorPkg =
        if observability != null && observability ? package then observability.package else pkgs.vector;

      # Core system tools always needed by the container runtime
      baseContents = [
        shellPkg
        pkgs.cacert
      ]
      ++ (
        if harden then
          [ ]
        else
          [
            pkgs.coreutils
            pkgs.procps
            pkgs.iproute2
            pkgs.gnused
          ]
      )
      ++ (pkgs.lib.optional (observability != null) vectorPkg)
      ++ (pkgs.lib.optional (observability != null) pkgs.coreutils);

      # Vector configuration
      vectorConfig =
        if observability != null then
          pkgs.writeText "vector.toml" ''
            data_dir = "/tmp/vector"

            [sources.logs]
            type = "file"
            include = ["/var/log/*.log", "/data/conduit/LOG"]
            fingerprint.strategy = "device_and_inode"

            [transforms.tag_logs]
            type = "remap"
            inputs = ["logs"]
            source = "if starts_with(.file, \"/data/conduit/\") {\n  .service = \"conduit\"\n} else {\n  m, err = parse_regex(.file, r'/var/log/(?P<service>[^/]+)\\.log$')\n  if err == null { .service = m.service }\n}"

            [sources.host_metrics]
            type = "host_metrics"
            collectors = ["cpu", "memory", "network", "filesystem", "load", "host"]

            [sinks.grafana_loki]
            type = "loki"
            inputs = ["tag_logs"]
            endpoint = "${observability.lokiUrl}"
            labels.app = "${name}"
            healthcheck.enabled = false

            [sinks.grafana_loki.encoding]
            codec = "json"

            [sinks.grafana_loki.auth]
            strategy = "basic"
            user = "${observability.lokiUser or "1644516"}"
            password = "''${GRAFANA_API_KEY}"

            [sinks.grafana_prometheus]
            type = "prometheus_remote_write"
            inputs = ["host_metrics"]
            endpoint = "${observability.prometheusUrl}"
            healthcheck.enabled = false

            [sinks.grafana_prometheus.auth]
            strategy = "basic"
            user = "${observability.prometheusUser or "3297682"}"
            password = "''${GRAFANA_API_KEY}"
          ''
        else
          null;

      # Create a derivation that puts the config in /etc/vector/vector.toml
      vectorConfigDerivation =
        if vectorConfig != null then
          pkgs.runCommand "vector-config" { } ''
            mkdir -p $out/etc/vector
            cp ${vectorConfig} $out/etc/vector/vector.toml
          ''
        else
          null;

      # Setup wrapper script
      entrypointScript = pkgs.writeShellScriptBin "entrypoint" ''
        set -e
        # Set container hostname to the configured image name
        echo "${name}" > /etc/hostname
        ${pkgs.hostname}/bin/hostname -F /etc/hostname || echo "⚠️ Could not set hostname"

        echo "=== Starting Container: ${name} ==="
        echo "Included Packages:"
        ${pkgs.lib.concatMapStringsSep "\n" (p: "echo \"  - ${p.name or "unknown"}\"") packages}
        ${pkgs.lib.optionalString (observability != null) "echo \"  - ${vectorPkg.name or "vector"}\""}
        echo "=================================="

        ${
          if vectorConfig != null then
            ''
               # Create directories for Vector and logs
               mkdir -p /tmp/vector
               mkdir -p /var/log

               # Generate runtime Vector config with the API key injected directly
               if [ -n "''$GRAFANA_API_KEY" ]; then
                 while IFS= read -r line || [ -n "''$line" ]; do
                   modified_line="''${line//\''${GRAFANA_API_KEY\}/''$GRAFANA_API_KEY}"
                   echo "''$modified_line"
                 done < /etc/vector/vector.toml > /tmp/vector.toml
               else
                 cp /etc/vector/vector.toml /tmp/vector.toml
               fi

               # Start Vector in the background (its own logs go to stdout/stderr so they are visible via flyctl logs)
              ${vectorPkg}/bin/vector --config /tmp/vector.toml 2>&1 | tee -a /var/log/vector.log & 
            ''
          else
            ""
        }
        ${entrypoint}
      '';

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
        ++ (pkgs.lib.optional (vectorConfigDerivation != null) vectorConfigDerivation)
        ++ packages
        ++ extraContents;

      config = {
        Entrypoint = [ "${entrypointScript}/bin/entrypoint" ];
        ExposedPorts = exposedPorts;
        Env = finalEnv;
      };
    };
}
