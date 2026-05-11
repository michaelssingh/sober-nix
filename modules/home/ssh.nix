{ pkgs, ... }:
{
  programs.ssh = {
    enable = true;
    enableDefaultConfig = false;

    matchBlocks = {
      "*" = {
        extraOptions = {
          AddKeysToAgent = "yes";
          IgnoreUnknown = "UseKeychain";
          IdentityAgent = "~/.ssh/agent.sock";
        };
      };

      "github.com" = {
        hostname = "github.com";
        user = "git";
      };
    };
  };

  services.ssh-agent.enable = true;
}
