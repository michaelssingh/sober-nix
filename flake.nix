{
  description = "SOBER Systems Infrastructure";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-25.11";
    home-manager = {
      url = "github:nix-community/home-manager/release-25.11";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    # gemini-cli.nvim isn't packaged in nixpkgs atm
    # fetch it and build manually in modules/nvim

    gemini-nvim = {
      url = "github:marcinjahn/gemini-cli.nvim";
      flake = false;
    };

    ani-cli = {
      url = "github:pystardust/ani-cli";
      flake = false;
    };

    nixpkgs-pinned = {
      url = "github:nixos/nixpkgs/4a3fc4cf736b7d2d288d7a8bf775ac8d4c0920b4";
    };

    nixpkgs-unstable = {
      url = "github:nixos/nixpkgs/nixos-unstable";
    };

    sops-nix = {
      url = "github:Mic92/sops-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    senpai-src = {
      url = "git+https://git.sr.ht/~delthas/senpai?ref=master";
      flake = false;
    };

    senpai-dev-src = {
      url = "git+file:///home/michael/git/sober-nix/packages/senpai?ref=dev-michael";
      flake = false;
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      nixpkgs-unstable,
      home-manager,
      sops-nix,
      ...
    }@inputs:
    let
      user = "michael";
      pkgs = nixpkgs.legacyPackages."x86_64-linux";
      overlays = import ./modules/overlays { inherit inputs; };
    in
    {
      nixosConfigurations = {
        otus = nixpkgs.lib.nixosSystem {
          specialArgs = { inherit inputs user; };

          modules = [
            sops-nix.nixosModules.sops
            ./hosts/otus/hardware-configuration.nix
            ./hosts/otus/default.nix

            {
              nixpkgs.overlays = [
                overlays.additions
                overlays.modifications
              ];
            }

            home-manager.nixosModules.home-manager
            {
              home-manager.useGlobalPkgs = true;
              home-manager.useUserPackages = true;
              home-manager.extraSpecialArgs = { inherit inputs user; };
              home-manager.users.${user} = import ./users/${user}/workstation.nix;
            }
          ];
        };
      };

      packages."x86_64-linux" = {
        athene-image = import ./hosts/athene/athene.nix { inherit pkgs; lib = nixpkgs.lib; };
        glaucidium-image = import ./hosts/glaucidium/glaucidium.nix { inherit pkgs; lib = nixpkgs.lib; };
        styx-image = import ./hosts/styx/styx.nix { inherit pkgs; lib = nixpkgs.lib; };
      };

      homeConfigurations = {
        server = home-manager.lib.homeManagerConfiguration {
          inherit pkgs;
          extraSpecialArgs = { inherit inputs user; };
          modules = [ ./users/${user}/server.nix ];
        };
      };
      # --- The DevShell for repo maintenance ---
      devShells."x86_64-linux".default = pkgs.mkShell {
        name = "dev";

        nativeBuildInputs = with pkgs; [
          gh # GitHub CLI
          git # Git for version control
          skopeo # Daemonless container image management
          nixfmt-rfc-style # Keep the config pretty
          openssl # Required by ani-cli
          dmidecode # Hardware info
          pciutils # PCI bus info
          usbutils # USB bus info
          smartmontools # Storage health checks
          btop # Resource monitoring
          iotop # Disk I/O monitoring
          iproute2 # Networking
          tcpdump # Packet capture
          nmap # Network audit
          ripgrep # Search
          fd # Find files
          jq # JSON processing
          dig
          powertop # Power management
          sops
          age
        ];
      };
    };
}
