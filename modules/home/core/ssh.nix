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
        identityFile = "~/.ssh/github";
      };

      "eu.nixbuild.net" = {
        hostname = "eu.nixbuild.net";
        user = "michael";
        identityFile = "~/.ssh/nixbuild";
        extraOptions = {
          PubkeyAcceptedKeyTypes = "ssh-ed25519";
          ServerAliveInterval = "60";
          RequestTTY = "no";
          Compression = "yes";
          ControlMaster = "auto";
          ControlPath = "/tmp/ssh-%r@%h:%p";
          ControlPersist = "10m";
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
        hostname = "fdaa:3:7a15:a7b:4d:9be5:53d9:2";
        port = 2222;
        user = "root";
        identityFile = "~/.ssh/fly";
      };
    };
  };

  # Use the systemd-managed ssh-agent
  services.ssh-agent.enable = true;
}
