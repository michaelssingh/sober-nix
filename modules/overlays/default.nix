{ inputs, ... }:
{
  # This 'additions' overlay adds new packages not in nixpkgs
  additions = final: prev: import ./gemini.nix { inherit inputs; } final prev;

  # This 'modifications' overlay changes existing packages
  modifications = final: prev: {
    # Example: pinning an existing package to a specific version
    # coreutils = prev.coreutils.override { ... };
  };
}
