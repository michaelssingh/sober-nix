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
