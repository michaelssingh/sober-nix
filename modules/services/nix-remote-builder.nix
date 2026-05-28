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

    programs.ssh.extraConfig = ''
      Host sober-styx.internal
        HostName fdaa:3:7a15:a7b:4d:9be5:53d9:2
        Port 2222
        User root
        StrictHostKeyChecking no
        UserKnownHostsFile /dev/null
    '';

    nix = {
      distributedBuilds = true;
      
      # Prefer remote builds over local ones
      settings.builders-use-substitutes = true;
      
      buildMachines = [
        {
          hostName = "sober-styx.internal";
          system = "x86_64-linux";
          protocol = "ssh-ng";
          maxJobs = 4;
          speedFactor = 2; 
          supportedFeatures = [ "nixos-test" "benchmark" "big-parallel" "kvm" ];
          mandatoryFeatures = [ ];
        }
      ];
    };
  };
}
