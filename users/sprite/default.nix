{
  pkgs,
  inputs,
  lib,
  config,
  ...
}:

let
  publicKeys = import ../../lib/public-keys.nix;
in
{
  imports = [
    inputs.sops-nix.homeManagerModules.sops
    ../michael/minimal.nix
    ../../modules/home/core/sober.nix
  ];

  # Apply overlays for custom packages (like 'antigravity' and 'sprite' CLI overrides)
  nixpkgs.overlays = [
    (import ../../modules/overlays { inherit inputs; }).additions
    (import ../../modules/overlays { inherit inputs; }).modifications
  ];

  # Allow unfree packages (needed for the sprite CLI)
  nixpkgs.config.allowUnfree = true;

  # Global Sober System Options
  sober.isRemote = true;

  home.username = "sprite";
  home.homeDirectory = "/home/sprite";
  home.stateVersion = "25.11";

  programs.home-manager.enable = true;

  # Sops-Nix Key Source for Home-Manager
  sops.age.keyFile = "/home/sprite/.config/sops/age/keys.txt";
  sops.defaultSopsFile = ../../secrets/secrets.yaml;

  home.packages = with pkgs; [
    git
    htop
    ripgrep
    fd
    jq
    antigravity
    sprite
    cachix
  ];

  # Declarative antigravity CLI Authentication
  sops.secrets.antigravity_oauth_token = {
    path = "/home/sprite/.gemini/antigravity-cli/antigravity-oauth-token";
  };

  # Declarative Cachix CLI Authentication
  sops.secrets.cachix_auth_token = { };

  # --- Declarative Git config to override platform defaults ---
  home.file.".gitconfig".text = ''
    [user]
      name = "Michael S. Singh"
      email = "michael@sober.fyi"
  '';

  # --- Declarative SSHD config ---
  home.file."sshd_config".text = ''
    Port 2222
    AuthorizedKeysFile .ssh/authorized_keys
    StrictModes no
    UsePAM yes
    UseDNS no
    GSSAPIAuthentication no
    Subsystem sftp /usr/lib/openssh/sftp-server
    PidFile /home/sprite/sshd.pid
  '';

  # --- Declarative SSH Authorized Keys ---
  home.file.".ssh/authorized_keys".text = ''
    ${publicKeys.forge}
    ${publicKeys.fly}
    ${publicKeys.nixbuild}
    ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPIQCZc54poJ8vqawd8TraNryQeJnvH1eLpIDgbiqymM
  '';

  # --- Declarative Nix configuration for the daemon ---
  home.file.".nix.conf_source".text = ''
    sandbox = true
    build-users-group = nixbld
    trusted-users = root sprite
    experimental-features = nix-command flakes
    max-jobs = 8
    cores = 0
    builders-use-substitutes = true

    # Cachix Cache Configurations
    substituters = https://cache.nixos.org https://sober-nix.cachix.org
    trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY= sober-nix.cachix.org-1:5txyMvuBOEoGah9zLW5SHrdLD92/h7eMiSv3VkErSG4=
  '';

  # --- Declarative SSH Agent Bridge Script wrapper to avoid comma-parsing bugs in sprite-env ---
  home.file.".ssh-agent-bridge.sh" = {
    executable = true;
    text = ''
      #!/usr/bin/env bash
      exec ${pkgs.socat}/bin/socat UNIX-LISTEN:/home/sprite/.ssh-agent.sock,fork,unlink-early TCP-CONNECT:127.0.0.1:9000
    '';
  };

  # --- Declarative SSHD Wrapper Script to ensure /run/sshd is created on boot ---
  home.file.".sshd-wrapper.sh" = {
    executable = true;
    text = ''
      #!/usr/bin/env bash
      /usr/bin/mkdir -p /run/sshd
      exec /usr/sbin/sshd -D -e -f /home/sprite/sshd_config
    '';
  };

  # --- Sprite Services Setup Activation Hook ---
  home.activation.configure-sprite-environment = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
        # Export /usr/bin to PATH so that sprite-env can invoke 'curl' and 'jq'
        export PATH="$PATH:/usr/bin"

        # 0. Configure Cachix auth token
        $DRY_RUN_CMD mkdir -p /home/sprite/.config/cachix
        if [ -f "${config.sops.secrets.cachix_auth_token.path}" ]; then
          token=$(cat "${config.sops.secrets.cachix_auth_token.path}")
          $DRY_RUN_CMD tee /home/sprite/.config/cachix/cachix.dhall <<EOF >/dev/null
    { authToken =
        "$token"
    , hostname = "https://cachix.org"
    , binaryCaches = [] : List { name : Text, secretKey : Text }
    }
    EOF
          $DRY_RUN_CMD chmod 600 /home/sprite/.config/cachix/cachix.dhall
        fi

        # 1. Update /etc/nix/nix.conf
        $DRY_RUN_CMD /usr/bin/sudo mkdir -p /etc/nix
        $DRY_RUN_CMD /usr/bin/sudo cp ${config.home.file.".nix.conf_source".source} /etc/nix/nix.conf
        $DRY_RUN_CMD /usr/bin/sudo chmod 644 /etc/nix/nix.conf

        # 2. Register nix-daemon as a Sprite service if not present
        if ! /.sprite/bin/sprite-env services list | grep -q "nix-daemon"; then
          $DRY_RUN_CMD /.sprite/bin/sprite-env services create nix-daemon \
            --cmd /usr/bin/sudo \
            --args "/nix/var/nix/profiles/default/bin/nix-daemon" \
            --no-stream
        fi
        $DRY_RUN_CMD /.sprite/bin/sprite-env services start nix-daemon || true

        # 3. Register sshd as a Sprite service if not present, or recreate if legacy command is found
        $DRY_RUN_CMD /usr/bin/sudo mkdir -p /run/sshd
        if /.sprite/bin/sprite-env services list | grep -q "sshd" && ! /.sprite/bin/sprite-env services list | grep -q ".sshd-wrapper.sh"; then
          $DRY_RUN_CMD /.sprite/bin/sprite-env services delete sshd
        fi
        if ! /.sprite/bin/sprite-env services list | grep -q "sshd"; then
          $DRY_RUN_CMD /.sprite/bin/sprite-env services create sshd \
            --cmd /usr/bin/sudo \
            --args "/home/sprite/.sshd-wrapper.sh" \
            --no-stream
        fi
        $DRY_RUN_CMD /.sprite/bin/sprite-env services start sshd || true

        # 4. Register ssh-agent-bridge as a Sprite service if not present
        if ! /.sprite/bin/sprite-env services list | grep -q "ssh-agent-bridge"; then
          $DRY_RUN_CMD /.sprite/bin/sprite-env services create ssh-agent-bridge \
            --cmd "/home/sprite/.ssh-agent-bridge.sh" \
            --no-stream
        fi
        $DRY_RUN_CMD /.sprite/bin/sprite-env services start ssh-agent-bridge || true

        # 5. Set default login shell to fish
        if [ "$(getent passwd sprite | cut -d: -f7)" != "/usr/bin/fish" ]; then
          $DRY_RUN_CMD /usr/bin/sudo /usr/bin/chsh -s /usr/bin/fish sprite
        fi
  '';
}
