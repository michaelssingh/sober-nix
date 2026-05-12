{ pkgs, ... }:
{
  programs.ssh = {
    enable = true;

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

      "sober-builder.internal" = {
        hostname = "sober-builder.internal";
        port = 2222;
        user = "root";
      };
    };
  };

  # Use the systemd-managed ssh-agent
  services.ssh-agent.enable = true;
}
