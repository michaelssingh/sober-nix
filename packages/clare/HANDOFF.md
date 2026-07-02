# Clare TUI — Agent Handoff Prompt

> **Context**: `clare` is a Go TUI anime streaming client at `packages/clare/` in the `sober-nix` NixOS flake repository. It uses Bubble Tea, lipgloss, and launches `mpv` for playback. This document describes all recent upgrades and remaining work.

---

## Architecture Overview

| File | Purpose |
|---|---|
| `main.go` | CLI flags, non-interactive mode, `tea.NewProgram` setup |
| `tui.go` | Bubble Tea model, Update loop, View rendering (~1850 lines) |
| `client.go` | AllAnime GraphQL API, stream resolution, AniSkip v2 API, stream cache + prefetch |
| `player.go` | `mpv` command builder, Lua script embedding, AniSkip chapters, resume positions |
| `history.go` | Watch history (`history.json`), positions (`positions.json`), data types |
| `cache.go` | Show/episode cache, Jikan metadata cache, image rendering (disabled) |
| `save-position.lua` | Embedded Lua script for mpv — saves position every 15s, marks episodes completed at 80% |
| `clare_test.go` | Unit + integration tests |

### TUI States
```
stateHistory → stateSearchInput → stateSearchRunning → stateShowSelect → stateEpisodeSelect → statePlaybackPreparing → statePlaybackActive
                                                                                              ↕
                                                                                          stateLogs (tab 3)
```

### Key Data Structures
- `AnimeShow.AvailableEpisodes` — `map[string]any` with keys `"sub"`, `"dub"`, `"raw"` mapping to `float64` episode counts
- `PositionsData` — `map[malID]ShowState` where `ShowState` has `ResumeState` (current position) and `CompletedEpisodes` (list of completed ep numbers)
- `HistoryEntry` — `{ShowID, ShowName, Episode, Timestamp}`
- `streamCache` — thread-safe `map[cacheKey]resolvedURL` for prefetched stream URLs

---

## Recent Upgrades (All Committed & Verified)

### 1. Dedicated Logs Tab with Live Streaming
- Third tab (`Logs [3]`) in top navigation, toggled with key `3` or navigated to via tab keys.
- Background goroutine `tailLogFile()` follows `~/.local/state/clare/debug.log` and pipes lines into `clareLogChan`.
- Single `clareLogMsg` handler in Update loop processes all log lines (both clare diagnostics and MPV output).

### 2. Always-On Logging
- Removed `CLARE_DEBUG` environment variable gate from `debugLog()` in `main.go`.
- All HTTP requests/responses, lifecycle events, MPV stdout/stderr, and state transitions are always logged to `debug.log`.
- **README.md is outdated** — still references `CLARE_DEBUG=1`. Needs updating.

### 3. Unified MPV Log Streaming
- MPV stdout and stderr are piped through `debugLog("[MPV] %s", ...)` instead of a separate `mpvLogChan`.
- Removed `MpvLogMsg` type and `readLogsCmd` — everything flows through the single `clareLogChan` → `clareLogMsg` path.
- MPV progress lines (`[MPV] AV:`) overwrite in-place in the viewport instead of creating new lines.

### 4. Smart Autoscrolling
- Viewport calls `GotoBottom()` only when `AtBottom()` is true.
- Users can scroll up to freeze the viewport and read previous logs while new logs continue appending.

### 5. Keypress Log Filtering
- `TUI KeyMsg:` lines are silently dropped in the `clareLogMsg` handler before being added to telemetry.
- They still write to `debug.log` for forensic debugging.

### 6. Full Mouse Support
- `tea.WithMouseCellMotion()` added to `tea.NewProgram` in `main.go`.
- `tea.MouseMsg` handler in Update dispatches to the appropriate component based on state (lists, viewport, etc.).
- Mouse scroll works in all views: history list, show list, episode list, and logs viewport.

### 7. Background Stream Prefetching
- `prefetchEpisodeStream()` in `client.go` resolves stream URLs in background goroutines.
- Thread-safe `streamCache` (`sync.RWMutex`) stores resolved URLs keyed by `showID-mode-epNo-quality`.
- `resolveStreamURL()` checks cache first (cache hit logged).
- Prefetch triggers:
  - On initial episode list load (`episodesResultMsg`) — prefetches selected + next episode.
  - On cursor movement in episode list — prefetches current + next episode.
- Deduplication via `activePrefetches` map prevents redundant concurrent fetches.

### 8. Completed Shows Filtering
- `refreshHistory()` in `tui.go` now loads `positions.json` and show cache.
- Shows are hidden from Continue Watching if:
  - The last watched episode number `>=` total episode count, OR
  - `CompletedEpisodes` count `>=` total episode count.
- Logged as `refreshHistory: hiding completed show ...`.

### 9. Temp Chapters File Race Fix
- `playbackFinishedMsg` struct now carries `tempLuaFile` and `tempChaptersFile` fields.
- `waitForExitCmd()` passes the temp paths that belong to *that specific process*.
- When killing a previous player in `resolvedPlaybackMsg`, old temp files are cleaned up immediately before starting the new process.
- This prevents a race where the new process's chapters file gets deleted by the old process's cleanup.

### 10. Cover Art Disabled
- `doFetchCoverArt()` returns `nil` — no more `chafa` subprocess calls or `renderImageANSI` log noise.
- The `coverArtCache` field and `CoverArtLoadedMsg` type still exist but are inert.
- The rendering infrastructure in `cache.go` (`renderImageANSI`, `downloadThumbnail`) is still present for future use.

### 11. HTTP Request/Response Logging
- `loggingRoundTripper` in `client.go` wraps `http.RoundTripper`.
- All HTTP clients log: `HTTP Request: GET <url>` and `HTTP Response: GET <url> -> Status 200`.

### 12. AniSkip v2 API Migration
- Migrated from `/api/v1/skip-times` to `/api/v2/skip-times/{malId}/{episodeNumber}`.
- Response format uses camelCase (`skipType`, `startTime`, `endTime`) instead of snake_case.
- Generates FFmpeg metadata chapters file for mpv's `--chapters-file` flag.

---

## Test Suite (11 Tests, All Passing)

| Test | What It Verifies |
|---|---|
| `TestCleanHTML` | HTML tag stripping from descriptions |
| `TestParseJikanDuration` | Duration string parsing |
| `TestParseEpisodeNumber` | Episode number extraction from strings |
| `TestPositionsFile` | Positions JSON save/load round-trip |
| `TestAniSkipAPI` | Live AniSkip v2 API call (skipped in Nix sandbox) |
| `TestChaptersFileGeneration` | FFmpeg chapters file generation from skip times |
| `TestLogStreamingAndFormatting` | Log file tailing and format consistency |
| `TestModelLoggingIntegration` | Bubble Tea model log dispatch and viewport update |
| `TestMpvProcessLoggingIntegration` | MPV subprocess log streaming through `debugLog` |
| `TestStreamCacheAndPrefetch` | Stream URL cache hit and prefetch deduplication |
| `TestCompletedShowsFiltering` | Completed shows hidden from history, incomplete shows shown |

---

## Remaining Work / Open Tasks

### Sub/Dub Availability Indicators (Not Started)
**Goal**: Make it clear in the TUI which episodes have subs, dubs, or both available.

**Available data**:
- `AnimeShow.AvailableEpisodes` is `map[string]any` with keys `"sub"`, `"dub"`, `"raw"` → `float64` episode counts.
- Example: `{"sub": 24, "dub": 12}` means episodes 1–24 have subs, episodes 1–12 have dubs.

**Suggested approach**:
- In the episode select header/title bar, show a badge like `[SUB+DUB]`, `[SUB only]`, or `[DUB only]` for the show overall.
- Per-episode: if the episode number exceeds the dub count, mark it `[SUB]`; if both are available, mark `[SUB+DUB]`.
- Use lipgloss styling: green for `SUB`, blue for `DUB`, purple for `SUB+DUB`.

### README.md Needs Updating
- Remove references to `CLARE_DEBUG=1` (logging is always on now).
- Add documentation for: Logs tab, mouse support, stream prefetching, completed show filtering.
- Add keyboard shortcut for tab `3` (Logs).

### Dead Code Cleanup (Optional)
- `coverArtCache` field, `CoverArtLoadedMsg` type, and all `doFetchCoverArt` call sites could be removed if cover art is permanently abandoned.
- `renderImageANSI` and `downloadThumbnail` in `cache.go` are unused.

---

## Build & Verification Commands

**Unit tests** (on remote VM):
```bash
cd packages/clare && go test -v .
```

**NixOS system build** (on remote VM):
```bash
NIX_REMOTE=daemon nix build .#nixosConfigurations.otus.config.system.build.toplevel \
  --no-link --extra-experimental-features 'nix-command flakes'
```

**Deploy to otus** (on otus):
```bash
git pull && sudo nixos-rebuild switch --flake .#otus
```

**Git commit rules**:
- Never bypass pre-commit hooks (`--no-verify` is prohibited).
- Sign commits: `SSH_AUTH_SOCK=/home/sprite/.ssh-agent.sock`
- Push to: `origin` (Forgejo at `sober-bubo.flycast:2222`)
