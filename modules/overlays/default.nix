{ inputs, ... }:
{
  # This 'additions' overlay adds new packages not in nixpkgs
  additions = final: prev: import ./gemini.nix { inherit inputs; } final prev;

  # This 'modifications' overlay changes existing packages
  modifications = final: prev: {
    # Example: pinning an existing package to a specific version
    # coreutils = prev.coreutils.override { ... };
    hydroxide = prev.hydroxide.overrideAttrs (old: rec {
      version = "0.2.31";
      src = final.fetchFromGitHub {
        owner = "emersion";
        repo = "hydroxide";
        rev = "v${version}";
        sha256 = "sha256-92eyt+s+kEXRuIXPRmbIQG5Mth7wJFCruqTN3wL5DhI=";
      };
      vendorHash = "sha256-CjvvVFjYRlykZwEqHtuD9qc/MsHZsJtKy2G6e2N7K0M=";
    });

    senpai = prev.senpai.overrideAttrs (oldAttrs: rec {
      version = "0.4.1";
      src = final.fetchFromGitHub {
        owner = "delthas";
        repo = "senpai";
        rev = "v${version}";
        sha256 = "1d16wbqm3hrydcb0308mg5cvgzz85vqq1bnwx0ly4647fr3f21wp";
      };
    });
  };
}
