# Clare ⚔️

`clare` is a lightweight, modular, and premium Go CLI client for streaming anime, movies, and TV series directly in your terminal, named after the Claymore protagonist. It replaces legacy TUI wrappers with a high-performance, asynchronous terminal interface built on the **Bubble Tea** framework.

---

## 🌟 Features

* **Vibrant Bubble Tea TUI:** Clean, interactive, event-driven user interface featuring a "Continue Watching" dashboard, multi-provider search results, episode lists, and real-time logs.
* **Multi-Provider Support:** Streamlined provider resolution supporting **AniDB** (Anime) and **VidSrc** (Movies & TV Series).
* **Episode Metadata & Kitsu Fallback:** Instant episode titles, synopses, and air dates via AniList and Jikan, with automatic failover to Kitsu API when rate-limited.
* **Airing Episode Filtering:** Automatically hides unreleased future episodes for currently airing series.
* **Persistent Disk Caching:** Local JSON caching under `~/.local/state/clare/cache/episodes/` to load titles instantly without network latency.
* **Local Playback Progress Resuming:** An embedded Lua script saves your time position to `~/.local/state/clare/positions.json` on pause/quit. Resumes automatically on next play.
* **Local Watch History Tracking:** Logs show details, watched episodes, and timestamps to `~/.local/state/clare/history.json` for the interactive "Continue Watching" dashboard.
* **Live Logs Tab:** Live-streams clare diagnostics and `mpv` process output to a dedicated scrollable log tab (`Logs [3]`).

---

## 📦 Installation & Setup

### 1. Ephemeral Run (Nix Run)
Run `clare` immediately without permanent installation using `nix run`:

```bash
# Run directly from local flake repository
nix run .#clare

# Run directly from private Forgejo Git forge
nix run git+ssh://git@sober-bubo.flycast/init/sober-nix.git#clare

# Run directly from GitHub mirror
nix run github:michaelssingh/sober-nix#clare
```

### 2. Declarative Installation (NixOS / Home Manager)
Add `pkgs.clare` to your Home Manager configuration (`modules/home/features/media-cli.nix`):

```nix
home.packages = [
  pkgs.clare
];
```

Then rebuild and apply your system configuration:

```bash
# On Ninox
sudo nixos-rebuild switch --flake .#ninox

# On Otus
sudo nixos-rebuild switch --flake .#otus
```

### 3. Imperative Nix Profile Install
Install directly into your user environment profile:

```bash
nix profile install .#clare
```

### 4. Build from Source (Standard Go Toolchain)
```bash
cd packages/clare
go build -o clare .
./clare
```

---

## 🚀 Keyboard Shortcuts

Within the interactive TUI:
* `1`, `2`, `3`, `4`: Quick tab switching between **Continue Watching**, **Search**, **Logs**, and **Config**.
* `s` or `/`: Open search input.
* `m`: Toggle audio translation mode (SUB vs DUB).
* `c`: Toggle inclusion of completed series in Continue Watching.
* `d`: Remove selected series from watch history.
* `Enter`: Select a show, fetch episodes, or play an episode.
* `Esc`: Go back to previous screen.
* `q` or `Ctrl+C`: Quit the application.

---

## 🔧 State & Logging

* **State Directory:** Defaults to `~/.local/state/clare`. Override with the `CLARE_STATE_DIR` environment variable.
* **Telemetry & Debug Logs:** Written continuously to `~/.local/state/clare/debug.log`:
  ```bash
  tail -f ~/.local/state/clare/debug.log
  ```
