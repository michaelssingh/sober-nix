# Clare ⚔️

`clare` is a lightweight, modular, and premium Go CLI client for streaming anime directly in your terminal, named after the Claymore protagonist. It replaces legacy bulky TUI wrappers with a high-performance, asynchronous terminal interface built on the **Bubble Tea** framework.

---

## 🌟 Features

* **Vibrant Bubble Tea TUI:** Clean, interactive, event-driven user interface featuring a "Continue Watching" dashboard, search results, episode lists, and real-time logs.
* **Dual Sub/Dub Audio Multiplexing:** Plays dubbed audio on top of subbed video (with hardcoded subtitles) dynamically. It queries the subbed stream track count ($N$) using `ffprobe` and maps the external dubbed stream as audio track $N+1$ in `mpv`.
* **Local Playback Progress Resuming:** An embedded Lua script saves your current time position to `~/.local/state/clare/positions.json` on pause/quit. Next time you play, it resumes instantly. Positions are automatically cleared when progress exceeds 95%.
* **Local Watch History Tracking:** Logs show details, watched episodes, and timestamps to `~/.local/state/clare/history.json` to feed the interactive "Continue Watching" menu. Includes filtering options to hide or show completed series.
* **Stream Prefetching:** Resolves stream URLs in the background for current and subsequent episodes based on list navigation, reducing playback delay.
* **Live Logs Tab:** Live-streams clare diagnostics and `mpv` process output to a dedicated scrollable log tab (`Logs [3]`).
* **Mouse Support:** Full cell-motion mouse support allows scrolling and selection clicks across lists and the log viewport.
* **Non-Interactive Scripting Mode:** Supports direct CLI playback (e.g. `clare -s "Fullmetal Alchemist: Brotherhood" -e 1`) bypassing the TUI entirely, useful for launcher bindings and shell scripting.
* **Offloaded Remote Compiles:** Designed to compile on high-powered remote builders and run locally on resource-constrained devices like `otus` with a tiny memory footprint (~10–15 MB RAM).

---

## 🚀 Keyboard Shortcuts

Within the interactive TUI:
* `1`, `2`, `3`: Quick tab switching between **Continue Watching**, **Search**, and **Logs**.
* `s` or `/`: Open search input (from Continue Watching or Search tabs).
* `c`: Toggle inclusion of completed series in the Continue Watching list.
* `d`: Remove the currently selected series from watch history.
* `Enter`: Select a show, fetch episodes, or play an episode.
* `Esc`: Go back to the previous screen or return to the search screen.
* `q` or `Ctrl+C`: Quit the application.

---

## 🔧 State & Logging

* **State Directory:** Defaults to `~/.local/state/clare`. Can be overridden with the `CLARE_STATE_DIR` environment variable.
* **Always-On Telemetry:** All network requests, lifecycle events, and `mpv` subprocess logs are continuously written to `~/.local/state/clare/debug.log`. You can stream these logs live within the TUI logs tab or using:
  ```bash
  tail -f ~/.local/state/clare/debug.log
  ```

