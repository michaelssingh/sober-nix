{ inputs, ... }:
{
  # This 'additions' overlay adds new packages not in nixpkgs
  additions = final: prev: 
    import ./gemini.nix { inherit inputs; } final prev // {
      tyto = final.callPackage ../../packages/tyto { };
      antigravity = final.callPackage ../../packages/antigravity { };
      strix-paste = final.callPackage ../../packages/strix-paste { };
      matrirc = final.callPackage ../../packages/matrirc { };
      unstable = import inputs.nixpkgs-unstable {
        system = final.stdenv.hostPlatform.system;
        config.allowUnfree = true;
      };
    };

  # This 'modifications' overlay changes existing packages
  modifications = final: prev: {
    # Example: pinning an existing package to a specific version
    # coreutils = prev.coreutils.override { ... };
    
    senpai = prev.senpai.overrideAttrs (old: {
      version = "0.5.0";
      src = final.fetchFromSourcehut {
        owner = "~delthas";
        repo = "senpai";
        rev = "v0.5.0";
        sha256 = "sha256-VjXgKdy4IpBhAP6uw/NtlexPki7nJzQi/HuY/+5lE/o=";
      };
      vendorHash = "sha256-4Ax9YVa9z1Unk3Z2iy9ZEqKjNmdgK0aF4GrD9ucXtjk=";
      ldflags = [
        "-X git.sr.ht/~delthas/senpai.version=v0.5.0"
      ];
    });

    neomutt = prev.neomutt.overrideAttrs (old: {
      version = "20260504";
      src = inputs.neomutt-src;
    });

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
