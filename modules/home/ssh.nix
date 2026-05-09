{ pkgs, ... }:
{
  programs.ssh = {
    enable = true;
    enableDefaultConfig = false;

    # Global settings for all hosts
    extraConfig = ''
      AddKeysToAgent yes
      IgnoreUnknown UseKeychain
    '';

    # Define your specific host aliases (MatchBlocks)
    matchBlocks = {
      "github.com" = {
        hostname = "github.com";
        user = "git";
        identityFile = "~/.ssh/id_ed25519_github_michaelssingh_athene";
      };

      # # Example for a future SOBER server or GCP instance
      # "sober-prod" = {
      #   hostname = "35.x.x.x"; # Your GCP IP later
      #   user = "michael";
      #   forwardAgent = true;
      # };
    };
  };
}
