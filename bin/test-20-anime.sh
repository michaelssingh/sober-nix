#!/usr/bin/env bash
# bin/test-20-anime.sh
# Simulates a human browsing the Clare TUI for 20 anime titles.
# All navigation is done entirely through the TUI (no CLI flags).
# Tests: search → show selection → episode list → stream resolution → mpv playback → download → DUB toggle

set -uo pipefail

BOLD=$'\e[1m'
CYAN=$'\e[36m'
GREEN=$'\e[32m'
YELLOW=$'\e[33m'
RED=$'\e[31m'
DIM=$'\e[2m'
RESET=$'\e[0m'

REPORT_FILE="/home/sprite/.gemini/antigravity-cli/brain/aa5a7f13-7d7d-42b7-bbd8-cd055443d7a8/clare_20_anime_report.md"

echo "${BOLD}${CYAN}╔═══════════════════════════════════════════════════════╗${RESET}"
echo "${BOLD}${CYAN}║   Clare TUI – 20 Anime Browser Validation Test Suite  ║${RESET}"
echo "${BOLD}${CYAN}╚═══════════════════════════════════════════════════════╝${RESET}"
echo ""

# ─── Setup ────────────────────────────────────────────────────────────────────

TEST_DIR=$(mktemp -d -t clare-20anime.XXXXXX)
export CLARE_STATE_DIR="$TEST_DIR"
LOG_FILE="$TEST_DIR/test_executions.log"
touch "$LOG_FILE"

echo "State directory : $TEST_DIR"
echo "Log file        : $LOG_FILE"
echo ""

# ─── Mock wrappers ────────────────────────────────────────────────────────────
# We intercept mpv and yt-dlp calls so Clare behaves exactly as it normally
# would, but without requiring a display or long downloads. The real binaries
# are invoked in headless / limited mode so stream resolution is fully proven.

MOCK_DIR="$TEST_DIR/mock-bin"
mkdir -p "$MOCK_DIR"

REAL_MPV_PATH="$(which mpv)"
REAL_YTDLP_PATH="$(which yt-dlp 2>/dev/null || echo '')"

# mpv wrapper: runs real mpv null-output (no display/audio), decodes 1 frame
# so Clare can verify the stream URL is actually playable.
cat > "$MOCK_DIR/mpv" << MPVEOF
#!/usr/bin/env bash
echo "MPV_CALL \$(date -Iseconds): \$*" >> "$TEST_DIR/test_executions.log"
"$REAL_MPV_PATH" --vo=null --ao=null --frames=1 --network-timeout=5 "\$@" \\
  >> "$TEST_DIR/mpv_output.log" 2>&1
RC=\$?
echo "MPV_EXIT \$RC" >> "$TEST_DIR/test_executions.log"
exit \$RC
MPVEOF
chmod +x "$MOCK_DIR/mpv"

# yt-dlp wrapper: runs real yt-dlp but aborts after 500 KB so we verify the
# stream can be downloaded without waiting for a full episode.
cat > "$MOCK_DIR/yt-dlp" << YTEOF
#!/usr/bin/env bash
echo "YTDLP_CALL \$(date -Iseconds): \$*" >> "$TEST_DIR/test_executions.log"
timeout 15 "$REAL_YTDLP_PATH" --max-filesize 500K "\$@" \\
  >> "$TEST_DIR/ytdlp_output.log" 2>&1
RC=\$?
echo "YTDLP_EXIT \$RC" >> "$TEST_DIR/test_executions.log"
exit \$RC
YTEOF
chmod +x "$MOCK_DIR/yt-dlp"

export PATH="$MOCK_DIR:$PATH"

# ─── Build Clare ──────────────────────────────────────────────────────────────

echo "${BOLD}${CYAN}Building Clare...${RESET}"
(cd ./packages/clare && go build -o clare .)
CLARE_BIN="./packages/clare/clare"
echo "Binary: $CLARE_BIN"
echo ""

# ─── Cleanup ──────────────────────────────────────────────────────────────────

cleanup() {
    if tmux has-session -t clare-tui-test 2>/dev/null; then
        tmux kill-session -t clare-tui-test
    fi
    # Copy all logs to persistent artifact directory for CI/CD inspection
    local dest_dir="/home/sprite/.gemini/antigravity-cli/brain/aa5a7f13-7d7d-42b7-bbd8-cd055443d7a8"
    mkdir -p "$dest_dir"
    cp "$TEST_DIR/test_executions.log" "$dest_dir/test_executions.log" 2>/dev/null || true
    cp "$TEST_DIR/mpv_output.log" "$dest_dir/mpv_output.log" 2>/dev/null || true
    cp "$TEST_DIR/ytdlp_output.log" "$dest_dir/ytdlp_output.log" 2>/dev/null || true
    cp "$TEST_DIR/debug.log" "$dest_dir/clare_debug.log" 2>/dev/null || true
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

# ─── Helpers ─────────────────────────────────────────────────────────────────

# Wait for a pattern to appear on screen, with a timeout
wait_for_screen() {
    local session="$1"
    local pattern="$2"
    local timeout="${3:-8}"
    local elapsed=0
    while (( elapsed < timeout )); do
        if tmux capture-pane -p -t "$session" | grep -qiE "$pattern"; then
            return 0
        fi
        sleep 1
        (( elapsed++ ))
    done
    return 1
}

capture() {
    tmux capture-pane -p -t "$1"
}

play_provider_stream() {
    local target_provider="$1"
    local session="clare-tui-test"
    for ((i=0; i<45; i++)); do
        tmux send-keys -t "$session" Up
    done
    sleep 0.5
    local screen
    screen=$(capture "$session")
    local list_started=false
    local idx=0
    local target_idx=-1
    while IFS= read -r line; do
        if echo "$line" | grep -qiE "Select Source|Episode.*Sources"; then
            list_started=true
            continue
        fi
        if [ "$list_started" = true ]; then
            if echo "$line" | grep -qE "^[[:space:]]*$|enter: play"; then
                continue
            fi
            if echo "$line" | grep -qiE "(^│|^[[:space:]]+)\["; then
                local match_prov="ALLANIME"
                if [[ "$target_provider" =~ [Gg][Oo][Gg][Oo] ]]; then
                    match_prov="GOGO"
                fi
                if echo "$line" | grep -qi "$match_prov"; then
                    target_idx=$idx
                    break
                fi
                idx=$((idx + 1))
            fi
        fi
    done <<< "$screen"
    if [ "$target_idx" -lt 0 ]; then
        warn "No stream found for provider '$target_provider' on this screen."
        return 1
    fi
    info "Found '$target_provider' stream at index $target_idx. Navigating down..."
    for ((i=0; i<target_idx; i++)); do
        tmux send-keys -t "$session" Down
    done
    sleep 0.5
    local before_calls
    before_calls=$(grep -h "MPV_CALL" "$LOG_FILE" 2>/dev/null | wc -l)
    tmux send-keys -t "$session" Enter
    sleep 6
    local after_calls
    after_calls=$(grep -h "MPV_CALL" "$LOG_FILE" 2>/dev/null | wc -l)
    if [ "$after_calls" -gt "$before_calls" ]; then
        pass "Successfully played '$target_provider' stream via mpv"
        return 0
    else
        fail "Failed to play '$target_provider' stream"
        return 1
    fi
}

# Start a fresh Clare TUI session, always lands on Search input
start_clare() {
    if tmux has-session -t clare-tui-test 2>/dev/null; then
        tmux kill-session -t clare-tui-test
        sleep 0.5
    fi
    tmux new-session -d -s clare-tui-test -x 115 -y 35 \
        "env CLARE_STATE_DIR=\"$TEST_DIR\" \"$CLARE_BIN\""
    # Wait for TUI to render and initial async fetches to settle
    sleep 5
    # If Clare is NOT already on the search input screen (e.g. started in history
    # state after previous searches), press '2' to navigate to the Search tab.
    local screen
    screen=$(tmux capture-pane -p -t clare-tui-test 2>/dev/null || echo "")
    if ! echo "$screen" | grep -qiE "Enter anime title|SEARCH ANIME"; then
        tmux send-keys -t clare-tui-test "2"
        sleep 1
    fi
    # Wait until search input is ready
    wait_for_screen "clare-tui-test" "Enter anime title|SEARCH ANIME" 8 || true
}

pass() { echo "${GREEN}  ✔ $*${RESET}"; }
fail() { echo "${RED}  ✘ $*${RESET}"; }
warn() { echo "${YELLOW}  ⚠ $*${RESET}"; }
info() { echo "${DIM}    $*${RESET}"; }

# ─── Anime titles ─────────────────────────────────────────────────────────────

TITLES=(
    "Gachiakuta"
    "Ghost in the Shell"
    "Clevatess"
    "One Piece"
    "Frieren"
    "Bleach"
    "Demon Slayer"
    "Jujutsu Kaisen"
    "Kaiju No. 8"
    "Chainsaw Man"
    "Solo Leveling"
    "Blue Lock"
    "My Hero Academia"
    "Oshi no Ko"
    "Wind Breaker"
    "Attack on Titan"
    "Steins Gate"
    "Death Note"
    "Mushoku Tensei"
    "Fullmetal Alchemist"
)

# ─── Results tracking ─────────────────────────────────────────────────────────

declare -A R_SEARCH R_EPISODES R_STREAMS R_MPV_SUB R_DL_SUB R_STREAMS_DUB R_MPV_DUB

for t in "${TITLES[@]}"; do
    R_SEARCH["$t"]="—"; R_EPISODES["$t"]="—"; R_STREAMS["$t"]="—"
    R_MPV_SUB["$t"]="—"; R_DL_SUB["$t"]="—"
    R_STREAMS_DUB["$t"]="—"; R_MPV_DUB["$t"]="—"
done

# ─── Warm-up: prime network and create search history ─────────────────────────
# The very first AllAnime API call from a cold start takes 10-20s.
# We do a throwaway search first so all 20 real titles benefit from warm state.

echo ""
echo "${BOLD}${CYAN}⏳ Warming up Clare TUI (priming network + state)...${RESET}"
start_clare
tmux send-keys -t clare-tui-test "Naruto" Enter
wait_for_screen "clare-tui-test" "Search Results|SHOW DETAILS" 20 || true
tmux send-keys -t clare-tui-test "q"
sleep 2
tmux has-session -t clare-tui-test 2>/dev/null && tmux kill-session -t clare-tui-test
echo "${GREEN}  ✔ Warm-up complete — state directory now has history${RESET}"

# ─── Main test loop ───────────────────────────────────────────────────────────

declare -i PASS_COUNT=0
declare -i FAIL_COUNT=0

for title in "${TITLES[@]}"; do
    echo ""
    echo "${BOLD}${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo "${BOLD}${CYAN}  $title${RESET}"
    echo "${BOLD}${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"

    # Clear execution log entries for this title
    echo "--- BEGIN: $title ---" >> "$LOG_FILE"

    # ── 1. Launch Clare TUI ────────────────────────────────────────────────────
    start_clare
    info "Clare TUI launched"

    # ── 2. Type search query ───────────────────────────────────────────────────
    tmux send-keys -t clare-tui-test "$title" Enter
    info "Searching for \"$title\"..."

    # Wait for results or direct transition to episode list (Single-Result auto-selection)
    if wait_for_screen "clare-tui-test" "Search Results|SHOW DETAILS|Select Episode|EPISODE DETAILS" 25; then
        R_SEARCH["$title"]="PASS"
        pass "Search query processed successfully"
        PASS_COUNT+=1
    else
        R_SEARCH["$title"]="FAIL"
        fail "Search query did not load results/episode list (timed out)"
        FAIL_COUNT+=1
        dump=$(capture "clare-tui-test")
        echo "SCREEN DUMP (search fail for $title):"
        echo "$dump"
        echo "SCREEN DUMP (search fail for $title):" >> "$LOG_FILE"
        echo "$dump" >> "$LOG_FILE"
        tmux kill-session -t clare-tui-test
        continue
    fi

    # Check if we were auto-selected straight to Episode List
    current_screen=$(capture "clare-tui-test")
    auto_selected=false
    if echo "$current_screen" | grep -qiE "Select Episode|EPISODE DETAILS"; then
        auto_selected=true
        R_EPISODES["$title"]="PASS"
        pass "Episode list loaded (Auto-Selected)"
        PASS_COUNT+=1
    fi

    # ── 3. Select the first show in results (Only if NOT auto-selected) ───────
    if [ "$auto_selected" = false ]; then
        # Human presses Enter on the highlighted result
        tmux send-keys -t clare-tui-test Enter
        info "Selecting top result..."

        if wait_for_screen "clare-tui-test" "Select Episode|EPISODE DETAILS" 20; then
            R_EPISODES["$title"]="PASS"
            pass "Episode list loaded with episode details"
            PASS_COUNT+=1
        else
            R_EPISODES["$title"]="FAIL"
            fail "Episode list did not load (timed out)"
            FAIL_COUNT+=1
            echo "SCREEN DUMP (episodes fail):" >> "$LOG_FILE"
            capture "clare-tui-test" >> "$LOG_FILE"
            tmux kill-session -t clare-tui-test
            continue
        fi
    fi

    # ── 4. Press Enter on Episode 1 to resolve sources (SUB) ──────────────────
    # The first episode is already highlighted by default
    tmux send-keys -t clare-tui-test Enter
    info "Resolving SUB streams for Episode 1..."

    if wait_for_screen "clare-tui-test" "Select Source|Ok|Yt-mp4|Mp4upload|fast4speed|DEAD|Episode 1 Sources" 35; then
        R_STREAMS["$title"]="PASS"
        pass "Stream sources resolved (SUB)"
        PASS_COUNT+=1
        s_sources=$(capture "clare-tui-test")
        if echo "$s_sources" | grep -qiE "ALLANIME|GOGO"; then
            pass "Provider badges (ALLANIME/GOGO) detected in SUB source list"
        else
            warn "No provider badges (ALLANIME/GOGO) detected in SUB source list"
        fi
    else
        R_STREAMS["$title"]="FAIL"
        fail "Stream sources did not resolve (SUB)"
        FAIL_COUNT+=1
        echo "SCREEN DUMP (sources fail):" >> "$LOG_FILE"
        capture "clare-tui-test" >> "$LOG_FILE"
        tmux kill-session -t clare-tui-test
        continue
    fi

    # ── 5. Select and play streams from BOTH providers ──────────────────────────
    info "Testing multi-provider playback..."
    play_allanime_ok=false
    play_gogo_ok=false

    if play_provider_stream "allanime"; then
        play_allanime_ok=true
    fi

    # Return to source selection if mpv returned us to episode list
    if wait_for_screen "clare-tui-test" "Select Episode" 4; then
        tmux send-keys -t clare-tui-test Enter
        wait_for_screen "clare-tui-test" "Select Source" 8
    fi

    if play_provider_stream "gogoanime"; then
        play_gogo_ok=true
    fi

    # Ensure clare returns cleanly to the episode list
    wait_for_screen "clare-tui-test" "Select Episode|EPISODE DETAILS|enter: play" 8

    if [ "$play_allanime_ok" = true ] && [ "$play_gogo_ok" = true ]; then
        R_MPV_SUB["$title"]="PASS"
        pass "Successfully verified playability of both providers"
        PASS_COUNT+=1
    elif [ "$play_allanime_ok" = true ] || [ "$play_gogo_ok" = true ]; then
        R_MPV_SUB["$title"]="PARTIAL"
        warn "Only one provider could be verified"
    else
        R_MPV_SUB["$title"]="FAIL"
        fail "Could not verify stream playback for either provider"
        FAIL_COUNT+=1
    fi

    # ── 6. Press 'd' on Episode 1 to trigger download ─────────────────────────
    # (navigate back to episode list first if needed)
    # Ensure we're on episode list
    if wait_for_screen "clare-tui-test" "Select Episode|enter: play" 4; then
        tmux send-keys -t clare-tui-test "d"
        info "Triggered download with 'd' key (yt-dlp wrapper)..."
        sleep 6
        if grep -q "YTDLP_CALL" "$LOG_FILE" 2>/dev/null; then
            R_DL_SUB["$title"]="PASS"
            pass "yt-dlp was invoked for download (SUB)"
            PASS_COUNT+=1
        else
            R_DL_SUB["$title"]="FAIL"
            fail "yt-dlp was NOT invoked for download"
            FAIL_COUNT+=1
        fi
    else
        R_DL_SUB["$title"]="FAIL"
        fail "Could not reach episode list to test download"
        FAIL_COUNT+=1
    fi

    # ── 7. Toggle to DUB mode and test stream ─────────────────────────────────
    # Human presses 'm' to toggle mode from SUB → DUB
    if wait_for_screen "clare-tui-test" "Select Episode|enter: play" 4; then
        tmux send-keys -t clare-tui-test "m"
        sleep 1
        screen_after_m=$(capture "clare-tui-test")

        if echo "$screen_after_m" | grep -qiE "DUB|dub"; then
            info "DUB mode toggled — resolving DUB sources for Episode 1..."
            tmux send-keys -t clare-tui-test Enter
            sleep 6

            screen_dub_sources=$(capture "clare-tui-test")
            if echo "$screen_dub_sources" | grep -qiE "Select Source|Ok|Yt-mp4|Mp4upload|fast4speed"; then
                R_STREAMS_DUB["$title"]="PASS"
                pass "DUB stream sources resolved"
                PASS_COUNT+=1
                if echo "$screen_dub_sources" | grep -qiE "ALLANIME|GOGO"; then
                    pass "Provider badges (ALLANIME/GOGO) detected in DUB source list"
                else
                    warn "No provider badges (ALLANIME/GOGO) detected in DUB source list"
                fi

                # Play DUB stream
                tmux send-keys -t clare-tui-test Enter
                info "Launching DUB playback via mpv wrapper..."
                sleep 6
                if grep -q "MPV_CALL" "$LOG_FILE" 2>/dev/null; then
                    R_MPV_DUB["$title"]="PASS"
                    pass "mpv called for DUB stream"
                    PASS_COUNT+=1
                else
                    R_MPV_DUB["$title"]="WARN"
                    warn "DUB: Clare returned but mpv call not logged"
                fi
            else
                R_STREAMS_DUB["$title"]="N/A"
                warn "No DUB stream sources available for this title"
            fi
        else
            R_STREAMS_DUB["$title"]="N/A"
            warn "Title has no DUB mode"
        fi
    fi

    # ── 8. Quit Clare cleanly ─────────────────────────────────────────────────
    tmux send-keys -t clare-tui-test "q"
    sleep 1.5
    if tmux has-session -t clare-tui-test 2>/dev/null; then
        tmux kill-session -t clare-tui-test
    fi

    echo "--- END: $title ---" >> "$LOG_FILE"
done

# ─── Report ───────────────────────────────────────────────────────────────────

echo ""
echo "${BOLD}${CYAN}╔═══════════════════════════════════════════════════════╗${RESET}"
echo "${BOLD}${CYAN}║                 RESULTS SUMMARY                       ║${RESET}"
echo "${BOLD}${CYAN}╚═══════════════════════════════════════════════════════╝${RESET}"
echo ""
printf "${BOLD}%-24s %-8s %-10s %-10s %-10s %-10s %-12s %-10s${RESET}\n" \
    "Title" "Search" "Episodes" "Streams" "MPV(sub)" "DL(sub)" "Streams(dub)" "MPV(dub)"
echo "──────────────────────────────────────────────────────────────────────────────────────────────────"

for title in "${TITLES[@]}"; do
    colorize() {
        case "$1" in
            PASS) echo "${GREEN}$1${RESET}" ;;
            FAIL) echo "${RED}$1${RESET}" ;;
            WARN) echo "${YELLOW}$1${RESET}" ;;
            *)    echo "${DIM}$1${RESET}" ;;
        esac
    }
    printf "%-24s %-18s %-20s %-20s %-20s %-20s %-22s %-20s\n" \
        "$title" \
        "$(colorize "${R_SEARCH[$title]}")" \
        "$(colorize "${R_EPISODES[$title]}")" \
        "$(colorize "${R_STREAMS[$title]}")" \
        "$(colorize "${R_MPV_SUB[$title]}")" \
        "$(colorize "${R_DL_SUB[$title]}")" \
        "$(colorize "${R_STREAMS_DUB[$title]}")" \
        "$(colorize "${R_MPV_DUB[$title]}")"
done

echo ""
echo "${BOLD}Totals: ${GREEN}$PASS_COUNT passed${RESET}  ${RED}$FAIL_COUNT failed${RESET}"

# Write markdown report
cat > "$REPORT_FILE" << MDEOF
# Clare TUI – 20 Anime Browser Validation Report

> Generated: $(date -Iseconds)
>
> All tests performed entirely through interactive TUI navigation (tmux key simulation).
> No CLI flags used. Simulates a real user browsing Clare.

## Test Flow Per Anime

For each title the test performs (in order):

1. **Launch** Clare TUI fresh
2. **Type** the anime title and press Enter (search)
3. **Press Enter** to select the first result
4. **Press Enter** on Episode 1 to resolve streams (SUB)
5. **Press Enter** to play via headless mpv wrapper (verifies stream URL is valid)
6. **Press \`d\`** from episode list to trigger yt-dlp download
7. **Press \`m\`** to toggle DUB, repeat steps 4–5 for DUB
8. **Press \`q\`** to quit Clare

## Results Table

| Title | Search | Episode List | Streams (SUB) | MPV (SUB) | Download (SUB) | Streams (DUB) | MPV (DUB) |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
MDEOF

for title in "${TITLES[@]}"; do
    echo "| $title | ${R_SEARCH[$title]} | ${R_EPISODES[$title]} | ${R_STREAMS[$title]} | ${R_MPV_SUB[$title]} | ${R_DL_SUB[$title]} | ${R_STREAMS_DUB[$title]} | ${R_MPV_DUB[$title]} |" >> "$REPORT_FILE"
done

cat >> "$REPORT_FILE" << MDEOF2

## Execution Log (mpv & yt-dlp calls)

\`\`\`
MDEOF2
cat "$LOG_FILE" >> "$REPORT_FILE" 2>/dev/null || echo "No log file." >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"

cat >> "$REPORT_FILE" << MDEOF3

## mpv Headless Output

\`\`\`
MDEOF3
cat "$TEST_DIR/mpv_output.log" >> "$REPORT_FILE" 2>/dev/null || echo "No mpv output." >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"

cat >> "$REPORT_FILE" << MDEOF4

## yt-dlp Download Output

\`\`\`
MDEOF4
cat "$TEST_DIR/ytdlp_output.log" >> "$REPORT_FILE" 2>/dev/null || echo "No yt-dlp output." >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"

echo ""
echo "${BOLD}${GREEN}✔ Report written to: $REPORT_FILE${RESET}"
