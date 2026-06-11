{
  config,
  lib,
  ...
}:

let
  cfg = config.sober.services.observability;
in
{
  options = {
    sober.services.observability = {
      enable = lib.mkEnableOption "Observability stack via Vector";
      # Loki/Prometheus URLs
      lokiUrl = lib.mkOption { type = lib.types.str; };
      prometheusUrl = lib.mkOption { type = lib.types.str; };
      # Secret paths (injected by sops-nix)
      apiKeyFile = lib.mkOption { type = lib.types.path; };
    };
  };

  config = lib.mkIf cfg.enable {
    # This module provides the configuration schema.
    # The actual implementation of Vector injection resides in lib/default.nix
  };
}
