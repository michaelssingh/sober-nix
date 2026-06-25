# Clare ⚔️

`clare` is a lightweight, modular, and premium Go CLI client for streaming anime directly in your terminal, named after the Claymore protagonist. It replaces legacy bulky TUI wrappers with a high-performance, asynchronous terminal interface built on the **Bubble Tea** framework.

---

## 🌟 Features

* **Vibrant Bubble Tea TUI:** Clean, interactive, event-driven user interface featuring a "Continue Watching" dashboard, search results, and episode lists.
* **Dual Sub/Dub Audio Multiplexing:** Plays dubbed audio on top of subbed video (with hardcoded subtitles) dynamically. It queries the subbed stream track count ($N$) using `ffprobe` and maps the external dubbed stream as audio track $N+1$ in `mpv`.
* **Local Playback Progress Resuming:** An embedded Lua script saves your current time position to `~/.local/state/clare/positions.json` on pause/quit. Next time you play, it resumes instantly. Positions are automatically cleared when progress exceeds 95%.
* **Local Watch History Tracking:** Logs show details, watched episodes, and timestamps to `~/.local/state/clare/history.json` to feed the interactive "Continue Watching" menu.
* **Non-Interactive Scripting Mode:** Supports direct CLI playback (e.g. `clare -s "Fullmetal Alchemist: Brotherhood" -e 1`) bypassing the TUI entirely, useful for launcher bindings and shell scripting.
* **Senpai IRC Now-Playing `/np` Compatibility:** Leverages global MPRIS protocols so that playing anime via `clare` automatically shows up in `senpai` when typing `/np`.
* **Offloaded Remote Compiles:** Designed to compile on high-powered remote builders and run locally on resource-constrained devices like `otus` with a tiny memory footprint (~10–15 MB RAM).

---

## 🚀 Keyboard Shortcuts

Within the interactive TUI:
* `s` or `/`: Open search input.
* `Enter`: Select a show, fetch episodes, or play an episode.
* `Esc`: Go back to the previous screen or return to the search screen.
* `q` or `Ctrl+C`: Quit the application.

---

## 🔧 Environment Variables

You can customize `clare` using the following environment variables:
* `CLARE_STATE_DIR`: Overrides the default state directory path (defaults to `~/.local/state/clare`).
* `CLARE_DEBUG`: Set `CLARE_DEBUG=1` to enable verbose telemetry logging to `debug.log` in your state directory.

---

## 🛠️ Telemetry & Troubleshooting

If you encounter issues resolving streams or launching the media player, you can enable live tracing:
```bash
export CLARE_DEBUG=1
tail -f ~/.local/state/clare/debug.log
```
This logs keyboard events, state transitions, API payloads, resolved stream URLs, and `mpv` process initialization parameters.
