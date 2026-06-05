{ pkgs, ... }:

let
  tokyoNightCss = pkgs.writeText "theme-tokyo-night.css" ''
    :root {
      --is-dark-theme: true;

      /* Tokyo Night Palette */
      --bg: #1a1b26;
      --bg-dark: #16161e;
      --bg-highlight: #292e42;
      --terminal-black: #414868;
      --fg: #c0caf5;
      --fg-dark: #a9b1d6;
      --fg-light: #3b4261;
      --blue: #7aa2f7;
      --cyan: #7dcfff;
      --purple: #bb9af7;
      --orange: #ff9e64;
      --red: #f7768e;
      --green: #9ece6a;
      --yellow: #e0af68;

      /* Gitea/Forgejo mappings */
      --color-body: var(--bg);
      --color-box-body: var(--bg-dark);
      --color-box-header: var(--bg-highlight);
      --color-card: var(--bg-dark);
      --color-footer: var(--bg-dark);
      --color-navbar: var(--bg-dark);
      --color-text: var(--fg);
      --color-text-dark: var(--fg-dark);
      --color-text-light: var(--fg-light);
      --color-primary: var(--blue);
      --color-primary-hover: var(--cyan);
      --color-primary-active: var(--purple);
      --color-active: var(--blue);
      --color-border: var(--terminal-black);
      --color-input-background: var(--bg-highlight);
      --color-input-border: var(--terminal-black);
      --color-input-text: var(--fg);
      --color-markup-code-block: var(--bg-dark);
    }
  '';

  entrypoint = pkgs.writeShellScriptBin "entrypoint" ''
    set -e
    
    # Create directories for customization and data
    mkdir -p /data/forgejo/custom/public/css
    cp ${tokyoNightCss} /data/forgejo/custom/public/css/theme-tokyo-night.css
    
    # Ensure the git user owns the data folder
    chown -R git:git /data/forgejo

    # Dynamically set ROOT_URL if FLY_APP_NAME is present
    APP_NAME="''${FLY_APP_NAME:-sober-bubo}"
    export GITEA__server__ROOT_URL="https://$APP_NAME.fly.dev/"

    echo "Starting Forgejo Git Forge as git user..."
    exec ${pkgs.su-exec}/bin/su-exec git ${pkgs.forgejo}/bin/forgejo web
  '';
in
pkgs.dockerTools.buildLayeredImage {
  name = "sober-bubo";
  tag = "latest";

  contents = [
    pkgs.forgejo
    pkgs.su-exec
    pkgs.bash
    pkgs.coreutils
    pkgs.git
    pkgs.cacert
    entrypoint
    (pkgs.writeTextDir "etc/passwd" ''
      root:x:0:0::/root:/bin/bash
      git:x:1000:1000:Git User:/data/forgejo:/bin/bash
    '')
    (pkgs.writeTextDir "etc/group" ''
      root:x:0:
      git:x:1000:
    '')
  ];

  config = {
    Entrypoint = [ "${entrypoint}/bin/entrypoint" ];
    ExposedPorts = {
      "3000/tcp" = { };
      "2222/tcp" = { };
    };
    Env = [
      "PATH=/bin"
      "USER=git"
      "HOME=/data/forgejo"
      "GITEA_WORK_DIR=/data/forgejo"
      "GITEA__database__DB_TYPE=sqlite3"
      "GITEA__database__PATH=/data/forgejo/data/gitea.db"
      "GITEA__server__HTTP_PORT=3000"
      "GITEA__server__START_SSH_SERVER=true"
      "GITEA__server__SSH_PORT=2222"
      "GITEA__server__SSH_LISTEN_PORT=2222"
      "GITEA__security__INSTALL_LOCK=true"
      "GITEA__service__DISABLE_REGISTRATION=true"
      "GITEA__ui__THEMES=forgejo-auto,forgejo-light,forgejo-dark,tokyo-night"
      "GITEA__ui__DEFAULT_THEME=tokyo-night"
      "SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
    ];
  };
}
