{ inputs, ... }:
{
  # This 'additions' overlay adds new packages not in nixpkgs
  additions = final: prev: 
    import ./gemini.nix { inherit inputs; } final prev // {
      unstable = import inputs.nixpkgs-unstable {
        system = final.system;
        config.allowUnfree = true;
      };
    };

  # This 'modifications' overlay changes existing packages
  modifications = final: prev: {
    # Example: pinning an existing package to a specific version
    # coreutils = prev.coreutils.override { ... };
    
    senpai = prev.senpai.overrideAttrs (old: {
      version = "master-${inputs.senpai-src.shortRev or "latest"}";
      src = inputs.senpai-src;
      vendorHash = "sha256-4Ax9YVa9z1Unk3Z2iy9ZEqKjNmdgK0aF4GrD9ucXtjk=";
      ldflags = [
        "-X git.sr.ht/~delthas/senpai.version=master-${builtins.substring 0 7 (inputs.senpai-src.rev or "latest")}"
      ];
    });

    senpai-dev = prev.senpai.overrideAttrs (old: {
      pname = "senpai-dev";
      version = "dev-${inputs.senpai-dev-src.shortRev or "latest"}";
      src = inputs.senpai-dev-src;
      vendorHash = "sha256-4Ax9YVa9z1Unk3Z2iy9ZEqKjNmdgK0aF4GrD9ucXtjk=";
      postInstall = (old.postInstall or "") + ''
        mv $out/bin/senpai $out/bin/senpai-dev
      '';
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
