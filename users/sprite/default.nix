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
  ];

  # Declarative antigravity CLI Authentication
  sops.secrets.antigravity_oauth_token = {
    path = "/home/sprite/.gemini/antigravity-cli/antigravity-oauth-token";
  };

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
    build-users-group = nixbld
    trusted-users = root sprite
    experimental-features = nix-command flakes
    max-jobs = 8
    cores = 0
    builders-use-substitutes = true
  '';

  # --- Sprite Services Setup Activation Hook ---
  home.activation.configure-sprite-environment = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
    # Export /usr/bin to PATH so that sprite-env can invoke 'curl' and 'jq'
    export PATH="$PATH:/usr/bin"

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

    # 3. Register sshd as a Sprite service if not present
    if ! /.sprite/bin/sprite-env services list | grep -q "sshd"; then
      $DRY_RUN_CMD /usr/bin/sudo mkdir -p /run/sshd
      $DRY_RUN_CMD /.sprite/bin/sprite-env services create sshd \
        --cmd /usr/bin/sudo \
        --args "/usr/sbin/sshd,-D,-e,-f,/home/sprite/sshd_config" \
        --no-stream
    fi
    $DRY_RUN_CMD /.sprite/bin/sprite-env services start sshd || true
  '';
}
