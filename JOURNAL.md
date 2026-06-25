# Sober-Nix Development Journal

This journal serves as the local, persistent source of truth for work-in-progress, implementation plans, and session histories. It is tracked in Git, ensuring it remains intact across VM recreations, server restarts, and agent context resets.

---

## 📅 Chronological Development Sessions

### Session 6: 2026-06-25 18:00 (Clare Anime Client: Bubble Tea TUI & Dual Audio Multiplexing)
* **Host / Context**: Developed and compiled on remote VM `agy` (by Antigravity agent), deployed and activated on local workstation `otus` via SSH reverse tunnel.
* **Commits**:
  - `63f6b32` (Antigravity on `agy`): *docs(journal): update journal to document Session 6 details and add host locations*
  - `2e96da9` (Antigravity on `agy`): *fix(lua): resolve shell_escape nil error, update temp file prefix, and bump version to 0.1.2*
  - `7eba459` (Antigravity on `agy`): *style: refine TUI to Tokyonight theme, enable minimal list views, and add search/filtering for episodes*
  - `d60d081` (Antigravity on `agy`): *feat: implement Bubble Tea TUI, watch history, and mpv playback position tracking for clare*
* **Accomplishments**:
  - Replaced legacy FZF script with an interactive, event-driven Go TUI using the Bubble Tea framework.
  - Implemented automatic watch history tracking in `history.json` and playback positions resuming via custom embedded Lua script in `positions.json`.
  - Added dynamic English audio track detection using `ffprobe` (to count video's internal audio streams) and mapped the external dub stream to `--aid=N+1` in `mpv`.
  - Wrapped the `clare` package in Nix to prepend `ffmpeg` (for `ffprobe`), `yt-dlp`, and `mpv` to its path at runtime.
  - Custom-styled the interface to fit the Tokyonight Storm color theme and enabled minimal list filtering.

---

### Session 5: 2026-06-25 14:20 (Declarative SSHD Wrapper & Auto-Healing Privilege Separation)
* **Host / Context**: Developed on local workstation `otus` (by Michael S. Singh), built and deployed to remote VM `agy`.
* **Commits**:
  - `7cb5b83` (Michael S. Singh on `otus`): *docs(journal): document Session 5 detailing the declarative sshd wrapper fix*
  - `3b2e00d` (Michael S. Singh on `otus`): *config: add declarative sshd wrapper to prevent missing /run/sshd failures*
* **Accomplishments**:
  - Created a wrapper script `~/.sshd-wrapper.sh` that ensures `/run/sshd` exists before starting the SSH daemon.
  - Registered `~/.sshd-wrapper.sh` as the launch argument for the `sshd` Sprite Service in `sprite-env` via the Home Manager activation hook.
  - Added self-healing logic to the activation hook to automatically detect and recreate/upgrade legacy direct `sshd` service definitions in the remote VM environment.
  - Verified remote `nix-daemon` and `sshd` services are running, stable, and persistent.

---

### Session 4: 2026-06-25 17:40 (Environment Recovery & SSH Agent Bridge Automation)
* **Host / Context**: Recovered on remote VM `agy` (by Michael S. Singh / agent).
* **Commits**:
  - `c735853` (Michael S. Singh on `agy`): *docs(journal): populate chronological session logs mapping to git history*
  - `9895194` (Michael S. Singh on `agy`): *config(sprite): automate ssh-agent-bridge and document tunnel proxying*
* **Accomplishments**:
  - Moved `/run/sshd` privilege separation directory creation outside of the service-creation conditional block in [users/sprite/default.nix](file:///home/sprite/sober-nix/users/sprite/default.nix), ensuring it is created on every Home Manager activation (since `/run` is on tmpfs and gets wiped on boot).
  - Created a persistent, self-healing `ssh-agent-bridge` Sprite Service using a dedicated wrapper script `.ssh-agent-bridge.sh` to resolve `sprite-env` comma-parsing bugs.
  - Documented the port forwarding proxy architecture (SOCKS5 on 1080, SSH Agent on 9000, Nix copy on 2223) in [GEMINI.md](file:///home/sprite/sober-nix/GEMINI.md).
  - Initialized this development journal (`JOURNAL.md`).

---

### Session 3: 2026-06-25 13:53 - 14:24 (Ani-CLI Dub Selection & Video Stream Processing)
* **Host / Context**: Developed on local workstation `otus` (by Michael S. Singh), built on remote VM `agy` and deployed back to `otus`.
* **Commits**:
  - `d5aa958` (Michael S. Singh on `otus`): *pkg/ani-cli: fix duplicate line in patch causing syntax error*
  - `63bdd4d` (Michael S. Singh on `otus`): *pkg/ani-cli: fix malformed patch - regenerate from proper diff*
  - `2a541b4` (Michael S. Singh on `otus`): *fix(ani-cli): filter out duplicate SUB URLs from DUB links list*
  - `7a36f0f` (Michael S. Singh on `otus`): *feat(ani-cli): append -sober to script version for clear identification*
  - `7b24bfd` (Michael S. Singh on `otus`): *fix(ani-cli): fix dub audio track selection and save position on end-file*
* **Accomplishments**:
  - Patched `ani-cli` to append `-sober` to the version string.
  - Resolved duplicate/broken lines in patches that caused compilation failure.
  - Added support for selecting English dub audio tracks and passing secondary audio files to `mpv` using `--audio-file` flags.
  - Filtered duplicate sub/dub URLs from source listings.

---

### Session 2: 2026-06-25 13:47 (Codified Sprite VM Environment & Decoupled Tunnel Workflow)
* **Host / Context**: Developed on local workstation `otus` (by Michael S. Singh).
* **Commits**:
  - `eaa98f7` (Michael S. Singh on `otus`): *config(sprite): declaratively manage gitconfig using Michael's identity*
  - `0f3a24d` (Michael S. Singh on `otus`): *docs(gemini): document commit signing requirements for agent*
  - `65f46d7` (Michael S. Singh on `otus`): *config(glaucidium): correct primary region to ewr*
  - `7497749` (Michael S. Singh on `otus`): *config(sprite): declaratively set login shell to fish via HM activation hook*
  - `1cfc81a` (Michael S. Singh on `otus`): *config(sshd): disable DNS lookup and GSSAPI auth to speed up connection setup*
  - `f87e891` (Michael S. Singh on `otus`): *doc(gemini): document Sprite Services and VM Persistence protocol*
  - `b3c55bf` (Michael S. Singh on `otus`): *config(ssh): conditionally proxy flycast/internal hosts on remote VM*
  - `a68e942` (Michael S. Singh on `otus`): *config(sprite): add StrictModes no to sshd and IdentityFile to tunnel config*
  - `23a051a` (Michael S. Singh on `otus`): *config(sprite): codify VM setup, register services, revert tunnel/deploy to agy and clean up surnia*
* **Accomplishments**:
  - Reverted `bin/sprite-tunnel` and `bin/deploy.sh` to target VM `agy` instead of Fly.io.
  - Fully codified `users/sprite/default.nix` to handle custom package overlays, `sshd_config`, `.ssh/authorized_keys`, `/etc/nix/nix.conf` optimizations, and automated service registration hook.
  - Added conditional `ProxyCommand` logic using `socat` over the forwarded SOCKS5 proxy to route `.flycast` network traffic seamlessly from the VM.
  - Purged old `surnia` configuration files, flake targets, and waybar monitors.

---

### Session 1: 2026-06-24 18:37 - 2026-06-25 00:57 (Initial Remote Workflow and Surnia VM Sandbox)
* **Host / Context**: Developed on local workstation `otus` (by Michael S. Singh) and remote microVM `surnia` (on Fly.io).
* **Commits**:
  - `7211e11` (Michael S. Singh on `otus`/`surnia`): *config(surnia): configure nix remote builder with max-jobs=0, import sops-nix, and add antigravity package & oauth token secret*
  - `b930c6f` (Michael S. Singh on `otus`/`surnia`): *fix(surnia): persist nix store on /data volume, survives redeployments*
  - `585cafd` (Michael S. Singh on `otus`/`surnia`): *feat(surnia): wire up full remote dev workflow (tunnel, deploy, bootstrap)*
  - `e45513c` (Michael S. Singh on `otus`/`surnia`): *feat(surnia): add bin/surnia lifecycle manager, replace deploy.sh*
  - `7d73c51` (Michael S. Singh on `otus`/`surnia`): *fix(surnia): pin nix-user-chroot URL, fix Nix store detection*
  - `9782c2e` (Michael S. Singh on `otus`/`surnia`): *docs: use nixos-rebuild switch instead of nh os switch*
  - `bead440` (Michael S. Singh on `otus`/`surnia`): *feat(waybar): add python3 system pkg, fix hosts exec path and compact display*
  - `d29f138` (Michael S. Singh on `otus`/`surnia`): *fix(otus): use wheelNeedsPassword=false for passwordless sudo*
  - `394b6c0` (Michael S. Singh on `otus`/`surnia`): *fix(surnia): source nix profile inside chroot for bootstrap*
  - `4971841` (Michael S. Singh on `otus`/`surnia`): *secrets: add agy.key*
* **Accomplishments**:
  - Set up `surnia` as an ephemeral Nix remote builder and environment using a `chroot` setup.
  - Wrote deployment, tunnel, and bootstrap scripts to copy closures and keep connections alive.
  - Optimized local workstation (`otus`) sudo rules and Waybar displays.
