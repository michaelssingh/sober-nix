# Walkthrough: Repository Restructuring & Container Hardening

We successfully restructured the `sober-nix` repository layout to completely separate workstation logic from server/container logic. We implemented a unified, production-grade container image builder library and refactored all Fly.io configurations to use it.

## Changes Completed

### 1. Unified Container Builder: [lib/default.nix](file:///home/michael/git/sober-nix/lib/default.nix)
Implemented the `mkContainerImage` helper function with the following properties:
- **Flag-driven Hardening (`harden = true` by default)**: Strips out developmental/interactive shells (`bashInteractive`) and debug packages (like `coreutils` in the path). Standard setup actions (like `mkdir`, `chmod`, `chown`) are executed using absolute Nix store paths (e.g. `${pkgs.coreutils}/bin/mkdir`) so they are never exposed in `/bin` or `/usr/bin` within the runtime environment.
- **Git Hook support (`usrBinEnv = true` option)**: Restricts `/usr/bin/env` symlink generation exclusively to `sober-bubo` where Git hook shebangs need to locate `/usr/bin/env bash`.
- **Automatic Writable `/tmp`**: Places a sticky-bit permissioned `/tmp` (`1777`) into all base container environments to guarantee standard application runtimes have a writable temp location out of the box.
- **Consolidated Environment Layers (`pkgs.symlinkJoin`)**: Consolidates layout files and directory configurations into a single OCI layer to prevent exceeding the standard 124-layer limit.
- **Dynamic Bin Search Path**: Dynamically maps active packages to generate a correct, minimal `$PATH` inside the container environment using `pkgs.lib.makeBinPath`, preventing runtime command-not-found crashes.
- **Line Break and Shell Name Safety**: Corrected string mapping to prevent empty blank lines inside `/etc/passwd` and `/etc/group` (which emit warning logs in container utilities), and dynamically resolves the shell path (mapping `bash-interactive` to `bash` binaries).
- **Core Dump Git Isolation**: Added `core.*` to `.gitignore` to prevent committing memory core dumps that exhaust memory on resource-constrained microVMs during git pushes.

### 2. Decoupled Static Configs: [lib/public-keys.nix](file:///home/michael/git/sober-nix/lib/public-keys.nix)
- Centralized all system SSH and WireGuard keys as static Nix data, renaming the SSH key option from `github` to `forge` to generalize it since it is used on multiple Git hosts.
- Replaced the complex module-attribute extraction hacks in `flake.nix` and `styx.nix` with direct imports.
- Updated `modules/core/public-keys.nix` to load this static file.

### 3. Physical Directory Restructuring
Created dedicated namespaces under `hosts/`:
- **Workstations**: Moved `hosts/otus/` to `hosts/workstation/otus/`.
- **Servers**: Moved `hosts/athene/`, `hosts/bubo/`, `hosts/clare/`, `hosts/glaucidium/`, `hosts/strix/`, and `hosts/styx/` under `hosts/server/`.
- **Host entrypoint standardization**: Renamed all main container Nix files (e.g. `athene.nix`, `bubo.nix`) to `default.nix` inside their respective folders.

### 4. Code Migrations & Path Corrections
- Migrated all 6 server/container `default.nix` files to use `soberLib.mkContainerImage`, dramatically reducing boilerplate.
- Updated all `hosts/server/<host>/deploy.sh` script references to point to the repository root at three levels deep (`../../../`) instead of two.
- Fixed relative imports inside `hosts/workstation/otus/default.nix` to point to `../../../modules/` instead of `../../modules/`.
- Enabled persistent data volume mounts for the remote builder `sober-styx` inside `hosts/server/styx/fly.toml` to support overlay-based Nix store caching across restarts.

---

## Verification & Testing

### 1. Container Image Compilations
Verified that all OCI container images compile successfully using the new decoupled builder:
```bash
nix build .#athene-image
nix build .#bubo-image
nix build .#clare-image
nix build .#glaucidium-image
nix build .#strix-image
nix build .#styx-image
```
*Result:* All 6 builds completed successfully.

### 2. Workstation NixOS Build
Verified that the workstation configuration builds cleanly under the new structure:
```bash
nixos-rebuild build --flake .#otus
```
*Result:* Build completed successfully with no warnings or missing path errors.

### 3. Git Push & Remotes
Committed all layout restructuring and code changes and pushed them to both remotes:
- **Forgejo Remote (`origin`)**: `ssh://git@sober-bubo.internal:2222/init/sober-nix.git`
- **GitHub Remote (`github`)**: `git@github.com:michaelssingh/sober-nix.git`
*Result:* Both push commands completed successfully.
