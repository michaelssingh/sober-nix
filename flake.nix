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

      ninoxInstaller = nixpkgs.lib.nixosSystem {
        specialArgs = { inherit inputs user; };
        modules = [
          "${nixpkgs}/nixos/modules/installer/cd-dvd/installation-cd-minimal.nix"
          disko.nixosModules.disko
          {
            nixpkgs.hostPlatform = "x86_64-linux";
            isoImage.makeEfiBootable = true;
            isoImage.makeUsbBootable = true;
            networking.hostName = "ninox-installer";
            networking.networkmanager.enable = true;
            networking.networkmanager.ensureProfiles.profiles = {
              "OCA_Guests" = {
                connection = {
                  id = "OCA_Guests";
                  type = "wifi";
                  autoconnect = true;
                };
                wifi = {
                  mode = "infrastructure";
                  ssid = "OCA_Guests";
                };
                wifi-security = {
                  key-mgmt = "wpa-psk";
                  psk = "Iamaguest!";
                };
              };
            };
            nix.settings.experimental-features = [
              "nix-command"
              "flakes"
            ];
            services.openssh = {
              enable = true;
              settings = {
                PermitRootLogin = "yes";
                PasswordAuthentication = false;
              };
            };
            users.users.root.openssh.authorizedKeys.keys = [
              publicKeys.forge
              publicKeys.fly
              publicKeys.nixbuild
              publicKeys.agy
            ];
            users.users.nixos.openssh.authorizedKeys.keys = [
              publicKeys.forge
              publicKeys.fly
              publicKeys.nixbuild
              publicKeys.agy
            ];
            environment.systemPackages = with pkgs; [
              git
              disko
              sops
              age
              parted
              gptfdisk
              cryptsetup
              btrfs-progs
              (pkgs.writeShellScriptBin "install-ninox" ''
                set -euo pipefail
                echo "=== Starting Ninox NixOS Installation ==="
                TARGET_DISK="''${1:-/dev/nvme0n1}"
                echo "Target disk: $TARGET_DISK"

                if [ ! -b "$TARGET_DISK" ]; then
                  echo "Error: Block device $TARGET_DISK does not exist!"
                  exit 1
                fi

                read -p "WARNING: All data on $TARGET_DISK will be wiped. Continue? (y/N) " -n 1 -r
                echo
                if [[ $REPLY =~ ^[Yy]$ ]]; then
                  echo "=== Partitioning $TARGET_DISK ==="
                  sgdisk -Z "$TARGET_DISK"
                  sgdisk -n 1:0:+1G -t 1:ef00 -c 1:ESP "$TARGET_DISK"
                  sgdisk -n 2:0:0 -t 2:8309 -c 2:luks "$TARGET_DISK"

                  PART1="''${TARGET_DISK}p1"
                  PART2="''${TARGET_DISK}p2"
                  if [ ! -b "$PART1" ]; then
                    PART1="''${TARGET_DISK}1"
                    PART2="''${TARGET_DISK}2"
                  fi

                  echo "=== Formatting EFI System Partition ==="
                  mkfs.vfat -F32 -n ESP "$PART1"

                  echo "=== Setting up LUKS Encryption ==="
                  cryptsetup luksFormat "$PART2"
                  cryptsetup open "$PART2" crypted

                  echo "=== Formatting Btrfs & Subvolumes ==="
                  mkfs.btrfs -f -L ninox /dev/mapper/crypted
                  mount /dev/mapper/crypted /mnt
                  btrfs subvolume create /mnt/@root
                  btrfs subvolume create /mnt/@home
                  btrfs subvolume create /mnt/@nix
                  umount /mnt

                  echo "=== Mounting Target Filesystems ==="
                  mount -o compress=zstd,noatime,subvol=@root /dev/mapper/crypted /mnt
                  mkdir -p /mnt/{boot,home,nix}
                  mount "$PART1" /mnt/boot
                  mount -o compress=zstd,noatime,subvol=@home /dev/mapper/crypted /mnt/home
                  mount -o compress=zstd,noatime,subvol=@nix /dev/mapper/crypted /mnt/nix

                  echo "=== Running NixOS Installation ==="
                  nixos-install --flake github:michaelssingh/sober-nix#ninox --no-channel-copy

                  echo "=== Provisioning SOPS Age Key ==="
                  mkdir -p /mnt/home/michael/.config/sops/age
                  if [ -f /home/nixos/.config/sops/age/keys.txt ]; then
                    cp /home/nixos/.config/sops/age/keys.txt /mnt/home/michael/.config/sops/age/keys.txt
                    chmod 600 /mnt/home/michael/.config/sops/age/keys.txt
                    echo "✓ Copied SOPS age key from live environment."
                  fi

                  echo "=== Provisioning sober-nix repository ==="
                  mkdir -p /mnt/home/michael/git
                  if [ ! -d /mnt/home/michael/git/sober-nix ]; then
                    echo "Cloning sober-nix repository into /home/michael/git/sober-nix..."
                    git clone https://github.com/michaelssingh/sober-nix.git /mnt/home/michael/git/sober-nix
                    chown -R 1000:100 /mnt/home/michael/git 2>/dev/null || true
                    echo "✓ Cloned sober-nix repository."
                  fi

                  echo "=== User Password Setup ==="
                  echo "Initial default password for user 'michael' is set to 'nixos'."
                  read -p "Would you like to set a custom password for user 'michael' now? (y/N) " -n 1 -r
                  echo
                  if [[ $REPLY =~ ^[Yy]$ ]]; then
                    nixos-enter --root /mnt -- passwd michael
                  fi

                  echo "=== Ninox installation complete! You may now reboot. ==="
                else
                  echo "Installation cancelled."
                  exit 1
                fi
              '')
            ];
          }
        ];
      };
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

        ninox = nixpkgs.lib.nixosSystem {
          specialArgs = { inherit inputs user; };

          modules = [
            disko.nixosModules.disko
            sops-nix.nixosModules.sops
            ./hosts/workstation/ninox/default.nix

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
        glaucidium = nixpkgs.lib.nixosSystem {
          specialArgs = { inherit inputs user; };

          modules = [
            sops-nix.nixosModules.sops
            ./hosts/server/glaucidium/nixos.nix
            {
              nixpkgs.overlays = [
                overlays.additions
                overlays.modifications
              ];
            }
          ];
        };
        ninox-installer = ninoxInstaller;
      };

      packages."x86_64-linux" = rec {
        ninox-iso = ninoxInstaller.config.system.build.isoImage;
        appservice-mgr = pkgs.callPackage ./tools/appservice-mgr { };
        tyto = pkgs.callPackage ./packages/tyto { };
        clare = pkgs.callPackage ./packages/clare { };
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
