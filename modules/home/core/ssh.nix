{ config, pkgs, lib, ... }: {
  services.ssh-agent.enable = !config.sober.isRemote;

  programs.ssh = {
    enable = true;

    # In newer HM, enableDefaultConfig might be deprecated or unchanged, let's keep it if no warning
    enableDefaultConfig = false;

    settings =
      let
        proxyCommand = lib.optionalAttrs config.sober.isRemote {
          ProxyCommand = "${pkgs.socat}/bin/socat - SOCKS5:127.0.0.1:%h:%p,socksport=1080";
        };
      in {
        "*" = {
          AddKeysToAgent = "yes";
        };

        "github.com" = {
          HostName = "github.com";
          User = "git";
          IdentityFile = "~/.ssh/github";
        };

        "eu.nixbuild.net" = {
          HostName = "eu.nixbuild.net";
          IdentityFile = "~/.ssh/nixbuild";
          PubkeyAcceptedKeyTypes = "ssh-ed25519";
          ServerAliveInterval = "60";
          IPQoS = "throughput";
        };

        "sober-athene.flycast" = {
          HostName = "sober-athene.flycast";
          Port = "2222";
          User = "root";
          IdentityFile = "~/.ssh/fly";
          StrictHostKeyChecking = "no";
          UserKnownHostsFile = "/dev/null";
        } // proxyCommand;

        "sober-styx.flycast" = {
          HostName = "sober-styx.flycast";
          Port = "2222";
          User = "root";
          IdentityFile = "~/.ssh/fly";
          StrictHostKeyChecking = "no";
          UserKnownHostsFile = "/dev/null";
        } // proxyCommand;

        "sober-bubo.flycast" = {
          HostName = "sober-bubo.flycast";
          Port = "2222";
          User = "git";
          IdentityFile = "~/.ssh/github";
          AddressFamily = "inet6";
          StrictHostKeyChecking = "no";
          UserKnownHostsFile = "/dev/null";
          ConnectionAttempts = "10";
          ConnectTimeout = "10";
        } // proxyCommand;

        "sober-bubo.internal" = {
          HostName = "sober-bubo.internal";
          Port = "2222";
          User = "git";
          IdentityFile = "~/.ssh/github";
          AddressFamily = "inet6";
          StrictHostKeyChecking = "no";
          UserKnownHostsFile = "/dev/null";
          ConnectionAttempts = "10";
          ConnectTimeout = "10";
        } // proxyCommand;

        "sprite-vm" = {
          HostName = "127.0.0.1";
          Port = "2222";
          User = "sprite";
          IdentityFile = "~/.ssh/fly";
          StrictHostKeyChecking = "no";
          UserKnownHostsFile = "/dev/null";
        };

        "otus" = {
          HostName = "10.13.13.2";
          User = "michael";
        };

        "ninox" = {
          HostName = "10.13.13.3";
          User = "michael";
        };
      };
  };
}
