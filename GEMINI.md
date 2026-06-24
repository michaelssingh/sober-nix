# Gemini CLI Configuration

## Search Strategy
- ALWAYS prefer `nh search` for looking up packages in the current Nix environment.
- Only fallback to other tools (like `nix search`) if `nh` fails or is insufficient.

## Documentation Protocol
- **Proactive Updates**: I will autonomously identify when documentation requires updates.
- **GEMINI.md (Shared)**: Update for architectural changes, new conventions, and shared workflows.
- **Memory (Private)**: Update for project-specific notes, debugging history, and personal context.
- **What to update**: Any significant change to the system's "mental model" (e.g., new helper functions, modified data structures, or altered build flows).

## Build & Validation
- **Verification**: Use `nixos-rebuild build --flake .` locally for all configuration validations.
- **Switch**: Use `sudo nixos-rebuild switch --flake .#otus` on `otus` to apply changes.
- **Validation**: Never consider a task complete without a successful local build.

## Git & Branching Strategy
- `main`: Production branch. Only merge "known good" configurations here.
- `feat/*` / `config/*`: Use dedicated branches for iterative development.
- "Clean Tree" Rule: Always commit changes before running any build or validation command. Avoid "dirty" git trees to ensure reproducible builds.
- **NEVER Bypass Hooks**: Pre-commit hooks must NEVER be bypassed (`--no-verify` is prohibited). If hooks fail, fix the underlying issues in the development environment until they pass.


## YouTube & Streaming Conventions
... (remaining content)

### Subscriptions
- Subscriptions are managed declaratively in `modules/home/features/subscriptions.nix` as a Nix list of attribute sets.
- Do not use CSV files.
- Each entry should include `id`, `title`, and `tags` (as a list of strings).

### Live Streams
- Live stream sources are managed in `modules/home/features/livestreams.txt` in `Title|URL` format.
- Use the `live` Fish function for launching:
    - `live <name>`: Launches stream with video.
    - `live -a <name>`: Launches stream in audio-only mode.
- The `live` command uses `socat` via MPV's IPC socket to manage playback state.

### Newsboat Queueing
- Use `macro A` to pipe unread YouTube video URLs directly to the `queue-unread` Fish helper function.
- This ensures non-persistent, in-memory queueing that filters for valid YouTube video URLs.

## Email Address Management
- Personal and work email addresses should be defined as variables in Nix modules where they are needed:
    - `emailPersonal = "michaelssingh@protonmail.com";`
    - `emailWork = "michael@sober.fyi";`
- Use these variables consistently across configurations to ensure accurate and maintainable settings.

## Fly.io MicroVM Hosts
- Fly.io hosts are built as OCI Docker images using `dockerTools.buildLayeredImage` (defined in `flake.nix` under packages).
- **Strix**: Hosts a private `rustypaste` pastebin service. Paste creation is secured using the `AUTH_TOKEN` environment variable, which is set at runtime as a Fly.io secret.
- **Styx**: Hosts a private Nix remote builder service.
  - SSH runs on port `2222` with authorization configured for the Fly SSH key.
  - Nested sandboxing is disabled (`sandbox = false`) and the build users group is empty (`build-users-group =`) inside the container to work around lack of nested user namespace/cgroup support in the container environment.
  - Accessible via the Flycast host `sober-styx.flycast` over the Fly WireGuard interface, allowing the Fly Proxy to automatically start/scale the MicroVM on demand when an SSH connection is initiated by a Nix build.
- **Bubo**: Hosts a private Forgejo Git forge service.
  - SSH runs on port `2222` with authorization configured via SSH keys registered within Forgejo.
  - Accessible via `sober-bubo.flycast` (enabling proxy-based auto-starting and scaling on demand) and `sober-bubo.internal` (direct connection to the active VM instance). Exposing port `2222` in `fly.toml` is required for Flycast routing to work.

## Neovim AI / CodeCompanion Configuration
- **API Key Management**: Auth with the Gemini API is configured dynamically via SOPS secrets (`sops.secrets.gemini_api_key`), read from `~/.config/sops-nix/secrets/gemini_api_key` using Neovim's `cmd:` loader.
- **Model**: Default model configured for CodeCompanion strategies is `gemini-3.1-pro-preview`.

## Sprite CLI Configuration
- **Version Override**: Overridden to `v0.0.1-rc44` in `modules/overlays/default.nix` under `modifications` directly, rather than building custom package files.
- **API Token Management**: Auth with the `sprite` CLI is managed declaratively via SOPS secrets (`sops.secrets.sprites_api_token`). A Home-Manager activation hook automatically provisions this token from the SOPS-decrypted path, parses the organization prefix, and writes the configuration to `~/.sprites/sprites.json` with permissions `0600` on deployment.

## Remote Development Workflow (sprite.dev → otus)

> **Context**: The agent runs on a fast remote `sprite.dev` Ubuntu VM (hostname: `agy`), NOT on `otus`.
> The repository here is the same one deployed to `otus`. All editing and verification happens remotely;
> application happens locally on `otus` via a git pull and `sudo nixos-rebuild switch --flake .#otus`.

### How SSH Key Access Works
The user runs `bin/sprite-tunnel start` on `otus` before starting an agent session. This script:
1. Starts a **SOCKS5 proxy** on `localhost:1080` (Python, `bin/socks5-proxy.py`) for internet routing.
2. Uses `socat` to bridge the local `SSH_AUTH_SOCK` Unix socket to **TCP port 9000** on `localhost`.
3. Runs `sprite proxy -s agy 2222` to create a local TCP listener on port `2222` that tunnels through the sprite.dev service to this VM.
4. Establishes an **SSH reverse tunnel** into the VM, including **local port forwarding** for development:
   - `-L 5173:127.0.0.1:5173`: Frontend access.
   - `-L 4000:127.0.0.1:4000`: Backend API access.
   - `-R 1080:127.0.0.1:1080`: SOCKS5 proxy for remote internet access.
   - `-R 9000:127.0.0.1:9000`: SSH agent forwarding.

On the remote VM, `SSH_AUTH_SOCK` is pointed at a socket that forwards over TCP port 9000 back through
the tunnel to `otus`'s real ssh-agent. **Private keys never leave `otus`.**

Verify agent access at the start of any session with:
```bash
ssh-add -l
```

### Nix on the Remote VM (Ubuntu 25.10)
Nix is installed via the official multi-user installer. The daemon does not run as a systemd service
(Ubuntu's init system is not configured it). 

#### Remote VM Initialization Protocol (Agent Mitigation)
To mitigate VM restarts and connection/build issues, the agent must proactively check and start the required background daemons at the beginning of every session:

1. **SSH Daemon**: Check if `sshd` is running. If not, start it:
   ```bash
   sudo mkdir -p /run/sshd
   sudo /usr/sbin/sshd
   ```
2. **Nix Daemon**: Check if `nix-daemon` is running. If not, start it:
   ```bash
   sudo /nix/var/nix/profiles/default/bin/nix-daemon 2>/tmp/nix-daemon.log &
   sleep 3
   ```

Critical configuration in `/etc/nix/nix.conf` — must contain:
```
build-users-group = nixbld
trusted-users = root sprite

# Parallelism — use all 8 cores
max-jobs = 8
cores = 0

# Performance
builders-use-substitutes = true
```
Without `trusted-users = root sprite`, the daemon will reject store writes (`Trusted: 0`).
Without `max-jobs = 8`, Nix defaults to `max-jobs = 1`, leaving 7 cores idle.
If settings are missing, overwrite the file and restart the daemon.

The `nix` binary must be invoked with `NIX_REMOTE=daemon` prefix since the shell profile
(`/nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh`) does not set this reliably in
non-login shells. Verify the daemon is reachable and trusted:
```bash
NIX_REMOTE=daemon /nix/var/nix/profiles/default/bin/nix \
  --extra-experimental-features 'nix-command flakes' store ping
# Expected output: Trusted: 1
```

**Daemon lifecycle — suspend vs shutdown:**
- **Suspend/resume**: The daemon process survives in memory and resumes normally. No restart needed.
- **Full VM shutdown/restart**: The daemon dies and the socket is lost. Must be restarted at the
  start of the next session using the command above.
- **systemd is NOT the init system** on this VM (PID 1 is not systemd — it is a container runtime).
  Do not attempt `systemctl enable/start nix-daemon` — it will fail with "Host is down".

### Build & Verification Command (on Remote VM)
This replaces `nixos-rebuild build --flake .` which only works on NixOS hosts:
```bash
NIX_REMOTE=daemon /nix/var/nix/profiles/default/bin/nix build \
  .#nixosConfigurations.otus.config.system.build.toplevel \
  --no-link \
  --extra-experimental-features 'nix-command flakes'
```
Never consider a task complete without a successful build.

### Full Workflow: Edit → Verify → Push → Apply

**Step 1 — Edit** (remote VM): Make changes to Nix configuration files in the repository.

**Step 2 — Verify** (remote VM): Start the Nix daemon (see above) and run the build command above.

**Step 3 — Commit & Push** (remote VM): Follow the "Clean Tree" rule. The SSH agent forwarded from
`otus` authenticates git pushes to `sober-bubo.flycast`:
```bash
git add <files>
git commit -m "config: <description>"
git push origin main
```

**Step 4 — Apply** (on `otus`, by the user): Pull and switch:
```bash
git pull
sudo nixos-rebuild switch --flake .#otus
```
Alternatively, you can use the automated deployment script from `otus` to build remotely on the VM, copy the output, and switch configurations:
```bash
./bin/deploy.sh
```


## SaaS Engine Project
- Dedicated project directory: `projects/saas-engine/`.
- Development is managed by a dedicated team of subagents.
- Refer to `projects/saas-engine/TEAM.md` for role definitions and team operational standards.

### Important Caveats for Agents
- **Do NOT modify `otus`-specific low-end/resource optimizations** (e.g., `perf.lowend`,
  `distributedBuilds`, `nixbuild.net` build machines). These are intentional for `otus`'s hardware.
- **The remote VM is Ubuntu**, not NixOS. Do not use `nixos-rebuild`, `nh os switch`, or attempt
  to manage systemd units here.
- **The Nix daemon must be manually started** each agent session — there is no persistent systemd
  service managing it on this VM.
