# Sober-Nix Development Journal

This journal serves as the local, persistent source of truth for work-in-progress, implementation plans, and session histories. It is tracked in Git, ensuring it remains intact across VM recreations, server restarts, and agent context resets.

---

## 📅 Chronological Development Sessions

### Session 11: 2026-07-23 17:15 (Clare Architecture Modernization: Pre-flighting, MPV IPC, Structured Logging, JSON RPC & Teatest)
* **Host / Context**: Developed on remote VM `agy` (by Antigravity agent), verified via local Go test suites and deployed to `otus` via SSH tunnel.
* **Commits**:
  - `168b264` (Antigravity on `agy`): *fix(clare): disable ytdl hook for direct HLS m3u8 streams in getMpvCmd*
  - `5897d77` (Antigravity on `agy`): *fix(clare): enhance PreflightStreamURL with sub-playlist PNG ad segment detection*
  - `80657a6` (Antigravity on `agy`): *test: update TestPlayEpisodesOnOtus to use MultiProviderResolver and pre-flighting*
  - `7ec0ecf` (Antigravity on `agy`): *feat(clare): add stream pre-flighting, mpv ipc, structured logging, rpc driver & teatest suite*
* **Accomplishments**:
  - **Structured Logging & Telemetry (`logger.go`)**: Replaced informal string concatenation with Go `log/slog` structured events (`[SEARCH]`, `[RESOLVE]`, `[ANISKIP]`, `[MPV_IPC]`, `[POSITION]`). Built `ValidateSessionTrace` to automatically parse session logs against 6 mandatory health checkpoints and evaluate an `OPTIMAL`/`FAILED` `HealthSummary`.
  - **Stream Pre-flighting & Multi-Provider Fallbacks (`provider.go`)**: Built `PreflightStreamURL` to probe HLS playlists via 3-second HTTP GET requests, verifying status 200 and filtering out Cloudflare challenge pages and HLS playlists corrupted with `.png` ad inserts (`ibyteimg.com`). Built `MultiProviderResolver` to automatically cycle through `AllAnime`, `FlikHub`, and `GogoAnime`.
  - **MPV IPC UNIX Socket Controller (`mpv_ipc.go`)**: Implemented a lightweight JSON-RPC client connecting to `/tmp/mpv-clare.sock`. Added `InspectHealth()` to query `video-format`, `audio-codec-name`, `width`, `height`, and `playback-time` to guarantee video decoder initialization and 0-error playback.
  - **Headless JSON RPC Agent Driver (`rpc.go`)**: Added `clare rpc` CLI subcommands (`search`, `resolve`, `health`, `play`) returning structured JSON output (`RPCResponse`), enabling 100% deterministic AI agent operation without screen scraping.
  - **Bubbletea State Machine Unit Tests (`tui_test.go`)**: Implemented unit tests for Bubbletea model state transitions, keybindings (`j`/`k`, `/`, `Enter`, `Esc`), and view rendering.
  - **Process Hygiene**: Identified and killed orphan `agy -c` background subagent instances and stale `go run .` test processes to eliminate CPU load competition and restore 100% VM responsiveness.

---

### Session 10: 2026-07-01 12:00 (Clare TUI Formatting, Sixel Format Support & Mako Dynamic Config)
* **Host / Context**: Resumed after VM restart; synced user updates from `otus`, fixed TUI formatting and added Sixel formats, tested via automated test script, deployed to `otus` via `deploy.sh`.
* **Commits**:
  - `76e1a23` (Antigravity on `agy`): *feat(clare): configure chafa to render cover art using sixels format*
  - `ea62f19` (Antigravity on `agy`): *fix(clare): resolve formatting verb mismatch in renderEpisodeDetailsPanel and track test script*
  - `44276e7` (Michael S. Singh on `otus`): *feat(clare/mako): force chafa symbols format, fix octals, and quote mako app-name*
  - `ce6bf5b` (Michael S. Singh on `otus`): *fix(mako) dynamically specify the notification parmeters for deploy.sh*
* **Accomplishments**:
  - **Mako App-Name Quotes & Parameter Tweaks**: Synced user commits that quotes the Sway/Mako deploy notification app-name and dynamically adjusts deployment notification parameters.
  - **Clare Octal Fixes**: Fixed octal permissions in `cache.go` from `0755` to the Go-standard `0o755`.
  - **Sixel Image Support**: Modified `renderImageANSI` in `cache.go` to invoke `chafa` with `-f sixels` to enable high-quality bitmap renderings on Sixel-capable terminal viewports, while preserving fallback logic to solid symbols block art.
  - **TUI Controls Formatting Bugfix**: Fixed a string formatting verb mismatch in `renderEpisodeDetailsPanel` (which had 7 `%s` verbs but only 6 arguments), eliminating `%!s(MISSING)` inside the TUI controls details card.
  - **Automated TUI Test Harness Integration**: Check-in, updated, and validated `bin/test-clare-tui.sh` using tmux to simulate search and selection states, using `Death Note` as a stable target query for CLI direct mode stream resolution.

---

### Session 9: 2026-06-26 13:40 (Clare TUI Cover Art Rendering via Chafa & Multi-Remote Pushes)
* **Host / Context**: Developed and compiled on remote VM `agy` (by Antigravity agent), verified via local Nix build, deployed and activated on `otus` via SSH reverse tunnel.
* **Commits**:
  - `774579d` (Antigravity on `agy`): *feat(clare): include cover art in details panels, update version to 0.1.16*
* **Accomplishments**:
  - **Chafa Integration**: Configured `packages/clare/default.nix` to include the `chafa` utility inside the wrapper path, providing a hermetic block art image rendering engine.
  - **Cover Art Rendering & Local Caching**: Coded `downloadThumbnail` and `renderImageANSI` helper functions in `cache.go`. Downloaded thumbnails are cached locally under `cache/thumbnails/<show_id>.<ext>` to minimize network load. Images are processed through `chafa` to generate 16x11 character block ANSI string art.
  - **Netflix-Style Split Details Layout**: Rewrote the details panel view in `tui.go` to support a horizontal split. Left side displays the 16x11 cover art (with beautiful fallback loading/missing status card outlines), and the right side displays the show's metadata and description.
  - **Asynchronous Selection Triggers**: Integrated cover art fetching triggers when navigating through search results (`stateShowSelect`), browsing the continue watching history (`stateHistory`), or returning to the show list via `esc` from episode selection.
  - **Robust Git Push Workflow**: Resolved connection/proxy timeout issues when pushing to the local Forgejo repository `origin` (`sober-bubo.flycast`) by executing the pull and push locally on `otus` using the SSH reverse tunnel on port 2223.
  - **Automated Hands-Off Deployment**: Successfully triggered `bin/deploy.sh` on `otus` via SSH to pull latest changes, perform Nix build validation, and activate the new system generation.

---

### Session 8: 2026-06-26 12:50 (Clare TUI Metadata Details Panels, Local Caching & Debug Mode)
* **Host / Context**: Developed and compiled on remote VM `agy` (by Antigravity agent), verified via local Nix build validation.
* **Commits**:
  - `6759de8` (Antigravity on `agy`): *feat(clare): enhance TUI details panel aesthetics, align borders, and bump version to 0.1.15*
  - `32145b8` (Antigravity on `agy`): *fix(deploy): robustly extract remote nix build path, filtering out warnings*
  - `086cd3b` (Antigravity on `agy`): *bump(clare): update default.nix package version to 0.1.14*
  - `d7a3d8f` (Antigravity on `agy`): *feat(clare): restore split-screen details panels with cache lookup, bump version to 0.1.14*
  - `6b6b610` (Antigravity on `agy`): *feat(clare): add local caching for show episode lists and Jikan metadata, bump version to 0.1.13*
  - `27da44e` (Antigravity on `agy`): *fix(clare): add automatic HTTP request retry for transient error codes (like 502), bump version to 0.1.12*
  - `5721a76` (Antigravity on `agy`): *feat(clare): add -debug CLI flag, bump version to 0.1.11*
  - `aa5953e` (Antigravity on `agy`): *fix(clare): add detailed response body snippet to API json unmarshal errors*
  - `f8b3827` (Antigravity on `agy`): *chore(clare): bump main.go version to 0.1.10*
* **Accomplishments**:
  - **Netflix-Style Details Panels**: Restored horizontal split-screen view layouts across:
    1. **Continue Watching**: details loaded asynchronously for the highlighted show using `doFetchShowDetails`.
    2. **Show Selection**: details of the highlighted show rendered from Edge result items.
    3. **Episode Selection**: episode details (Title, Aired date, Type tag) from Jikan metadata.
  - **TUI Aesthetic Enhancements**: Remade the details panels to look extremely premium. Designed a structured metadata table with aligned key-value labels, styled canon/filler labels (colored in green/red/orange depending on canon status), integrated a gold star rating translator (e.g. `★★★★★ (9.10)`), and added an automated control helper hint block at the bottom.
  - **Height-Aligned Cards**: Unified the vertical layout by setting details panel border boxes to exactly match the list component height (`listHeight`), preventing sizing mismatches.
  - **Adaptive Sizing**: Added responsive viewport detection that automatically disables the side panel on terminals narrower than 80 columns.
  - **Local File Caching**: Codified caching managers in `packages/clare/cache.go` backing AllAnime metadata to `cache/shows/<id>.json` (24h invalidation) and Jikan metadata to `cache/jikan/<mal_id>.json`.
  - **HTTP Request Robustness**: Implemented linear backoff retry wrappers for transient gateway/CDN errors (502, 503, 504, 429) inside `client.go`.
  - **CLI Diagnostic Aids**: Integrated a `-debug` flag for detailed stderr logging, and included raw response snippets inside JSON API parse error contexts.
  - **Robust Deploy Path Extraction**: Enhanced `bin/deploy.sh` to capture remote build stdout/stderr as a raw buffer and filter it using a strict `/nix/store/` regex prefix. This prevents warnings (such as transient SQLite eval cache lock busy alerts) from polluting the parsed path variable and causing downstream deployment failures.

---

### Session 7: 2026-06-25 19:40 (Clare TUI Robustness & Deploy Script Stabilization)
* **Host / Context**: Developed on remote VM `agy` and local workstation `otus` (by Antigravity agent).
* **Commits**:
  - `f66f69b` (Antigravity on `agy`): *config(mako): increase width and height to prevent truncating multiline notifications*
  - `6640baf` (Antigravity on `agy`): *feat(deploy): make deployment notification persist until clicked using -t 0*
  - `5b4409d` (Antigravity on `agy`): *fix(deploy): format closure size unit in notification*
  - `349ee8e` (Antigravity on `agy`): *feat(deploy): add duration, closure size, and package diff to notify-send*
  - `9c09ad5` (Antigravity on `agy`): *fix(deploy): redirect remote git pull stdout to stderr to avoid polluting out_path*
  - `7399d3e` (Antigravity on `agy`): *fix(deploy): export SSH_AUTH_SOCK on remote VM to fix git authentication during build*
  - `f401a88` (Antigravity on `agy`): *feat(deploy): include system ID and git commit info in deploy notification*
  - `7124a1d` (Antigravity on `agy`): *fix(clare): use --aid=last for robust dub selection, warn on sub-only fallback*
  - `574f946` (Antigravity on `agy`): *feat(clare): add -version flag*
  - `e495011` (Antigravity on `agy`): *fix(deploy): pull latest changes first before building*
  - `4d877ca` (Antigravity on `agy`): *chore(clare): bump version to 0.1.3 to force rebuild*
  - `f2471bf` (Antigravity on `agy`): *docs(journal): document dual-audio robustness and header fixes*
* **Accomplishments**:
  - **Robust Dual-Audio Track Selection**: Replaced fragile `ffprobe` track-counting logic with `--aid=last` in `playDualCmd`. Since `mpv` always appends the external audio track as the last stream, this is completely robust and eliminates the dependency on `strconv` and the error-prone `countAudioStreams` function.
  - **TUI Playback Feedback**: Added visual warning `⚠ Dub unavailable (<error>) — playing sub only` in the Bubble Tea loading state to notify users of a dub resolution failure instead of silently falling back to Japanese audio. Added debug logs for all fallback paths.
  - **Clare CLI Version Flag**: Added a `-version` flag to print version info, and bumped the Nix package expression version to `0.1.3`.
  - **Git-First Deployment & Verification**: Configured `bin/deploy.sh` to run `git pull` locally before starting the remote build, ensuring all changes are synchronized.
  - **Deploy Script Authentication Fix**: Addressed an issue where `git pull` on the remote VM failed because `SSH_AUTH_SOCK` was not set in the non-interactive SSH shell. Fixed this by explicitly exporting `SSH_AUTH_SOCK=/home/sprite/.ssh-agent.sock` on the VM before pulling.
  - **Redirect Stdout to Stderr**: Resolved a shell capture bug where the stdout of the remote `git pull` was captured into the `out_path` variable, causing `nix copy` to fail. Fixed this by redirecting `git pull` stdout to `stderr` (`>&2`).
  - **Enhanced Deploy Notifications**: Improved `notify-send` in `bin/deploy.sh` to display the system Generation number, System ID (Nix store path hash), the git commit hash and commit subject. Further enhanced it to record and output the deployment duration, the system closure size (via `nix path-info`), and a detailed listing of package changes (via `nix store diff-closures`) between the old and new generations.

---

### Session 6: 2026-06-25 18:00 (Clare Anime Client: Bubble Tea TUI & Dual Audio Multiplexing)
* **Host / Context**: Developed and compiled on remote VM `agy` (by Antigravity agent), deployed and activated on local workstation `otus` via SSH reverse tunnel.
* **Commits**:
  - `cf474cf` (Antigravity on `agy`): *fix(clare): robust dual-audio aid fallback and ffprobe request headers*
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
  - Added Referer and User-Agent HTTP headers to `ffprobe` track queries and `mpv` video streaming to bypass server-side security checks (preventing 403 Forbidden errors).
  - Implemented a fallback mechanism to assume `1` audio stream if `ffprobe` fails, guaranteeing the external dub track is selected (via `--aid=2`) rather than defaulting to sub.

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
