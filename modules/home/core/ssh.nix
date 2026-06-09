{ pkgs, ... }:
{
  services.ssh-agent.enable = true;

  programs.ssh = {
    enable = true;
    
    # In newer HM, enableDefaultConfig might be deprecated or unchanged, let's keep it if no warning
    enableDefaultConfig = false;

    settings = {
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
      };

      "sober-styx.flycast" = {
        HostName = "sober-styx.flycast";
        Port = "2222";
        User = "root";
        IdentityFile = "~/.ssh/fly";
        StrictHostKeyChecking = "no";
        UserKnownHostsFile = "/dev/null";
      };

      "sober-bubo.fly.dev" = {
        HostName = "sober-bubo.fly.dev";
        Port = "2222";
        User = "git";
        IdentityFile = "~/.ssh/github";
        AddressFamily = "inet6";
        StrictHostKeyChecking = "no";
        UserKnownHostsFile = "/dev/null";
        ConnectionAttempts = "10";
        ConnectTimeout = "10";
      };
    };
  };
}
