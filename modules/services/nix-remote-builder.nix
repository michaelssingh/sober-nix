{ config, lib, ... }:

let
  cfg = config.sober.services.nix-remote-builder;
in
{
  # --- Documentation ---
  # This module is DORMANT by default. 
  # When 'enable = false', no configuration is applied, and no connections are attempted.
  # When 'enable = true', it configures the local Nix daemon to offload builds to 'sober-services'.

  options = {
    sober.services.nix-remote-builder = {
      enable = lib.mkEnableOption "Remote Nix Builders (Fly.io/Sober-Services)";
    };
  };

  config = lib.mkIf cfg.enable {
    # Safety Check: Ensure the transport layer (Fly WireGuard) is enabled
    assertions = [
      {
        assertion = config.sober.services.fly-wireguard.enable;
        message = "Remote builders require Fly.io WireGuard to be enabled for secure connectivity to 'sober-services.internal'.";
      }
    ];

    nix = {
      distributedBuilds = true;
      
      # Prefer remote builds over local ones
      settings.builders-use-substitutes = true;
      
      buildMachines = [
        {
          hostName = "sober-services.internal";
          sshUser = "root";
          system = config.nixpkgs.hostPlatform.system;
          protocol = "ssh-ng";
          maxJobs = 8;
          speedFactor = 1; # Reduced to be lower than nixbuild.net (default 1)
          supportedFeatures = [ "nixos-test" "benchmark" "big-parallel" "kvm" ];
          mandatoryFeatures = [ ];
        }
      ];
    };
  };
}
