{ pkgs, ... }:
{
  services.ssh-agent.enable = true;

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
        identityFile = "~/.ssh/github";
      };

      "eu.nixbuild.net" = {
        hostname = "eu.nixbuild.net";
        identityFile = "~/.ssh/nixbuild";
        extraOptions = {
          PubkeyAcceptedKeyTypes = "ssh-ed25519";
          ServerAliveInterval = "60";
          IPQoS = "throughput";
        };
      };

      "athene.internal" = {
        hostname = "fdaa:3:7a15:a7b:572:11c:754f:2";
        port = 2222;
        user = "root";
        identityFile = "~/.ssh/fly";
        extraOptions = {
          StrictHostKeyChecking = "no";
          UserKnownHostsFile = "/dev/null";
        };
      };

      "sober-styx.internal" = {
        hostname = "sober-styx.flycast";
        port = 2222;
        user = "root";
        identityFile = "~/.ssh/fly";
        extraOptions = {
          StrictHostKeyChecking = "no";
          UserKnownHostsFile = "/dev/null";
        };
      };
    };
  };
}
