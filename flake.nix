{
  description = "SOBER Systems Infrastructure";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-26.05";
    home-manager = {
      url = "github:nix-community/home-manager/release-26.05";
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

    nixpkgs-25_11 = {
      url = "github:nixos/nixpkgs/nixos-25.11";
    };

    sops-nix = {
      url = "github:Mic92/sops-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    disko = {
      url = "github:nix-community/disko";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    neomutt-src = {
      url = "github:neomutt/neomutt/20260504";
      flake = false;
    };
  };

  outputs =
    {
      nixpkgs,
      home-manager,
      sops-nix,
      disko,
      ...
    }@inputs:
    let
      user = "michael";
      pkgs = nixpkgs.legacyPackages."x86_64-linux";
      overlays = import ./modules/overlays { inherit inputs; };
      soberLib = import ./lib { inherit pkgs; };
      publicKeys = import ./lib/public-keys.nix;
    in
    {
      nixosConfigurations = {
        otus = nixpkgs.lib.nixosSystem {
          specialArgs = { inherit inputs user; };

          modules = [
            sops-nix.nixosModules.sops
            ./hosts/workstation/otus/hardware-configuration.nix
            ./hosts/workstation/otus/default.nix

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

        thinkpad = nixpkgs.lib.nixosSystem {
          specialArgs = { inherit inputs user; };

          modules = [
            disko.nixosModules.disko
            sops-nix.nixosModules.sops
            ./hosts/workstation/thinkpad/default.nix

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
        appservice-mgr = pkgs.callPackage ./tools/appservice-mgr { };
        tyto = pkgs.callPackage ./packages/tyto { };
        clare = pkgs.callPackage ./packages/clare { };
        ani-cli = pkgs.callPackage ./packages/ani-cli {
          src = inputs.ani-cli;
        };
        antigravity = pkgs.callPackage ./packages/antigravity { };
        strix-paste = pkgs.callPackage ./packages/strix-paste { };
        matrirc = pkgs.callPackage ./packages/matrirc { };
        athene-image = import ./hosts/server/athene {
          inherit pkgs soberLib inputs;
          inherit (nixpkgs) lib;
        };
        glaucidium-image = import ./hosts/server/glaucidium {
          inherit pkgs soberLib;
          inherit (nixpkgs) lib;
        };
        strix-image = import ./hosts/server/strix {
          inherit pkgs soberLib;
          inherit (nixpkgs) lib;
        };
        bubo-image = import ./hosts/server/bubo {
          inherit pkgs soberLib;
          inherit (nixpkgs) lib;
        };
        styx-image = import ./hosts/server/styx {
          inherit pkgs soberLib publicKeys;
          inherit (nixpkgs) lib;
        };
      };

      homeConfigurations = {
        server = home-manager.lib.homeManagerConfiguration {
          inherit pkgs;
          extraSpecialArgs = { inherit inputs user; };
          modules = [ ./users/${user}/server.nix ];
        };
        sprite = home-manager.lib.homeManagerConfiguration {
          inherit pkgs;
          extraSpecialArgs = {
            inherit inputs;
            user = "sprite";
          };
          modules = [ ./users/sprite/default.nix ];
        };
      };
      # --- The DevShell for repo maintenance ---
      devShells."x86_64-linux".default = pkgs.mkShell {
        name = "dev";

        nativeBuildInputs = with pkgs; [
          gh # GitHub CLI
          git # Git for version control
          skopeo # Daemonless container image management
          nixfmt # Keep the config pretty
          openssl # Required by ani-cli
          dmidecode # Hardware info
          pciutils # PCI bus info
          usbutils # USB bus info
          smartmontools # Storage health checks
          btop # Resource monitoring
          iotop # Disk I/O monitoring
          iproute2 # Networking
          go # Go programming language
          gopls # Go language server
          nodejs # Node.js and npm for package management

          tcpdump # Packet capture
          nmap # Network audit
          ripgrep # Search
          fd # Find files
          jq # JSON processing
          dig
          powertop # Power management
          sops
          age
          gitleaks # Secret scanning
          pre-commit # Git hooks framework
          deadnix # Find dead Nix code
          statix # Lints and suggestions for Nix code
          rocksdb
        ];
      };
    };
}
