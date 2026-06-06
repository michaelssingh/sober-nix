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

      "hashnix.club" = {
        hostname = "hashnix.club";
        user = "init";
        identityFile = "~/.ssh/hashnix";
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

      "sober-athene.flycast" = {
        hostname = "sober-athene.flycast";
        port = 2222;
        user = "root";
        identityFile = "~/.ssh/fly";
        extraOptions = {
          StrictHostKeyChecking = "no";
          UserKnownHostsFile = "/dev/null";
        };
      };

      "sober-styx.flycast" = {
        hostname = "sober-styx.flycast";
        port = 2222;
        user = "root";
        identityFile = "~/.ssh/fly";
        extraOptions = {
          StrictHostKeyChecking = "no";
          UserKnownHostsFile = "/dev/null";
        };
      };

      "sober-bubo.fly.dev" = {
        hostname = "sober-bubo.fly.dev";
        port = 2222;
        user = "git";
        identityFile = "~/.ssh/github";
        extraOptions = {
          AddressFamily = "inet6";
          StrictHostKeyChecking = "no";
          UserKnownHostsFile = "/dev/null";
        };
      };
    };
  };
}
