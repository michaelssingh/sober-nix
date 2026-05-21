{ pkgs, ... }:
{
  programs.ssh = {
    enable = true;
    enableDefaultConfig = false;

    matchBlocks = {
      "*" = {
        extraOptions = {
          AddKeysToAgent = "yes";
        };
      };

      "github.com" = {
        hostname = "github.com";
        user = "git";
      };

      "sober-services.internal" = {
        hostname = "sober-services.internal";
        port = 2222;
        user = "root";
      };
    };
  };

  # Use the systemd-managed ssh-agent
  services.ssh-agent.enable = true;
}
