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
- **Verification**: Use `nixos-rebuild build --flake . --build-host sober-services.internal` for all configuration validations.
- **Validation**: Never consider a task complete without a successful build on the remote host.

## Git & Branching Strategy
- **`main`**: Production branch. Only merge "known good" configurations here.
- **`feat/*` / `config/*`**: Use dedicated branches for iterative development.
- **"Clean Tree" Rule**: Always commit changes before running any build or validation command. Avoid "dirty" git trees to ensure reproducible builds.

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
