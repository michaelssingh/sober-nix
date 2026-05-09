{ pkgs, ... }:
{
  programs.ssh = {
    enable = true;
    enableDefaultConfig = false;

    # Global settings for all hosts
    extraConfig = ''
      AddKeysToAgent yes
      IgnoreUnknown UseKeychain
      IdentityAgent ~/.ssh/agent.sock
    '';

    # Define your specific host aliases (MatchBlocks)
    matchBlocks = {
      "github.com" = {
        hostname = "github.com";
        user = "git";
        identityFile = "~/.ssh/id_ed25519_github_michaelssingh_athene";
      };
    };
  };
}
