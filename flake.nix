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
  };

  outputs =
    {
      self,
      nixpkgs,
      home-manager,
      ...
    }@inputs:
    let
      user = "michael";
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      overlays = import ./modules/overlays { inherit inputs; };
    in
    {
      nixosConfigurations = {
        otus = nixpkgs.lib.nixosSystem {
          inherit system;

          specialArgs = { inherit inputs user; };

          modules = [
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
      # --- The DevShell for repo maintenance ---
      devShells.${system}.default = pkgs.mkShell {
        name = "sober-nix-dev";

        nativeBuildInputs = with pkgs; [
          gh # GitHub CLI
          git # Git for version control
          nixfmt-rfc-style # Keep the config pretty
        ];

        shellHook = ''
          echo "❄️ SOBER Systems Development Environment Loaded"
        '';
      };
    };
}
