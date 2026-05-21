{ inputs, ... }:
{
  # This 'additions' overlay adds new packages not in nixpkgs
  additions = final: prev: 
    import ./gemini.nix { inherit inputs; } final prev;

  # This 'modifications' overlay changes existing packages
  modifications = final: prev: {
    # Example: pinning an existing package to a specific version
    # coreutils = prev.coreutils.override { ... };
    hydroxide = prev.hydroxide.overrideAttrs (old: rec {
      version = "0.2.32";
      src = final.fetchFromGitHub {
        owner = "emersion";
        repo = "hydroxide";
        rev = "v${version}";
        sha256 = "sha256-3cSJkNTD5+L3VXO5I/1xo1tp9+H4/Z/tc2f8B63lGrc=";
      };
      vendorHash = "sha256-BIHvURCgqEzhl4NsVB7vBwLqMPxkM3CQgHmIcSTdOE4=";
    });
  };
}
