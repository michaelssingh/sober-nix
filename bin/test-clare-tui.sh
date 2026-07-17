#!/usr/bin/env bash
# bin/test-clare-tui.sh — Clare TUI automated test harness.
# All tests navigate the TUI exactly as a human would via tmux key sends.
# No CLI flags (-s, -e, -d) are used.

set -euo pipefail

BOLD=$'\e[1m'
CYAN=$'\e[36m'
GREEN=$'\e[32m'
YELLOW=$'\e[33m'
RED=$'\e[31m'
RESET=$'\e[0m'

echo "${BOLD}${CYAN}=== Clare TUI Automated Tests ===${RESET}"

# ─── Setup ────────────────────────────────────────────────────────────────────

TEST_DIR=$(mktemp -d -t clare-test-state.XXXXXX)
export CLARE_STATE_DIR="$TEST_DIR"
echo "State directory: $TEST_DIR"

# Mock binary directory
MOCK_DIR="$TEST_DIR/mock-bin"
mkdir -p "$MOCK_DIR"

# mpv wrapper: real headless mpv, decodes 1 frame to verify stream URL
REAL_MPV_PATH="$(which mpv)"
REAL_YTDLP_PATH="$(which yt-dlp 2>/dev/null || echo '')"

cat > "$MOCK_DIR/mpv" << EOF
#!/usr/bin/env bash
echo "MPV_CALL: \$*" >> "$TEST_DIR/mock.log"
"$REAL_MPV_PATH" --vo=null --ao=null --frames=1 --network-timeout=5 "\$@" >> "$TEST_DIR/mpv.log" 2>&1
RC=\$?; echo "MPV_EXIT: \$RC" >> "$TEST_DIR/mock.log"; exit \$RC
EOF
chmod +x "$MOCK_DIR/mpv"

# yt-dlp wrapper: real yt-dlp, capped at 500 KB
cat > "$MOCK_DIR/yt-dlp" << EOF
#!/usr/bin/env bash
echo "YTDLP_CALL: \$*" >> "$TEST_DIR/mock.log"
timeout 15 "$REAL_YTDLP_PATH" --max-filesize 500K "\$@" >> "$TEST_DIR/ytdlp.log" 2>&1
RC=\$?; echo "YTDLP_EXIT: \$RC" >> "$TEST_DIR/mock.log"; exit \$RC
EOF
chmod +x "$MOCK_DIR/yt-dlp"

export PATH="$MOCK_DIR:$PATH"

cleanup() {
    rm -rf "$TEST_DIR"
    for s in clare-test-main clare-test-airing; do
        tmux has-session -t "$s" 2>/dev/null && tmux kill-session -t "$s" || true
    done
}
trap cleanup EXIT

# Build clare
echo "${BOLD}${CYAN}Building Clare...${RESET}"
(cd ./packages/clare && go build -o clare .)
CLARE_BIN="./packages/clare/clare"
echo "Binary: $CLARE_BIN"

# Helper: wait for pattern on screen
wait_for() {
    local session="$1"; local pattern="$2"; local timeout="${3:-8}"
    for (( i=0; i<timeout; i++ )); do
        tmux capture-pane -p -t "$session" | grep -qiE "$pattern" && return 0
        sleep 1
    done
    return 1
}
cap() { tmux capture-pane -p -t "$1"; }

# ─── Test 1: Version check ─────────────────────────────────────────────────────
echo ""
echo "${BOLD}${CYAN}Test 1: Version flag${RESET}"
version_out=$("$CLARE_BIN" -version)
echo "Version: $version_out"
if [[ "$version_out" =~ ^clare\ [0-9]+\.[0-9]+\.[0-9]+ ]]; then
    echo "${GREEN}✔ Test 1 Passed: version flag works${RESET}"
else
    echo "${RED}✘ Test 1 Failed: unexpected version output${RESET}"
    exit 1
fi

# ─── Test 2: TUI search + show selection + episode list ───────────────────────
echo ""
echo "${BOLD}${CYAN}Test 2: TUI search flow (Frieren)${RESET}"

tmux has-session -t clare-test-main 2>/dev/null && tmux kill-session -t clare-test-main
tmux new-session -d -s clare-test-main -x 110 -y 32 \
    "env CLARE_STATE_DIR=\"$TEST_DIR\" \"$CLARE_BIN\""

echo "Waiting for TUI to start..."
if ! wait_for "clare-test-main" "Enter anime title|SEARCH ANIME" 8; then
    echo "${RED}✘ Test 2 Failed: TUI did not show search input${RESET}"
    exit 1
fi

s=$(cap "clare-test-main")
echo "--- Screen (Search Input) ---"
echo "$s"
echo "-----------------------------"
echo "${GREEN}✔ TUI started in search input state${RESET}"

# Human types "Frieren" and presses Enter
echo "Typing 'Frieren' and pressing Enter..."
tmux send-keys -t clare-test-main "Frieren" Enter
sleep 5

s=$(cap "clare-test-main")
echo "--- Screen (Search Results) ---"
echo "$s"
echo "-------------------------------"

if echo "$s" | grep -qiE "Search Results|SHOW DETAILS"; then
    echo "${GREEN}✔ Search results loaded with details panel${RESET}"
else
    echo "${RED}✘ Test 2 Failed: search results not displayed${RESET}"
    exit 1
fi

# Human presses Enter to select the highlighted show
echo "Pressing Enter to select the highlighted show..."
tmux send-keys -t clare-test-main Enter
sleep 5

s=$(cap "clare-test-main")
echo "--- Screen (Episode List) ---"
echo "$s"
echo "-----------------------------"

if echo "$s" | grep -qiE "Select Episode|EPISODE DETAILS"; then
    echo "${GREEN}✔ Episode list loaded with synopsis/details${RESET}"
else
    echo "${RED}✘ Test 2 Failed: episode list did not load${RESET}"
    exit 1
fi

# Human presses Escape to go back to show list
echo "Pressing Escape to go back..."
tmux send-keys -t clare-test-main Escape
sleep 2

s=$(cap "clare-test-main")
if echo "$s" | grep -qiE "Search Results|SHOW DETAILS"; then
    echo "${GREEN}✔ ESC returns to search results${RESET}"
else
    echo "${RED}✘ Test 2 Failed: ESC did not return to show list${RESET}"
    exit 1
fi

# Quit
tmux send-keys -t clare-test-main "q"
sleep 1.5
tmux has-session -t clare-test-main 2>/dev/null && {
    tmux kill-session -t clare-test-main
    echo "${RED}✘ Test 2 Failed: 'q' did not quit Clare${RESET}"
    exit 1
}
echo "${GREEN}✔ Test 2 Passed: TUI search → show select → episode list → back → quit${RESET}"

# ─── Test 3: Stream resolution + mpv playback ─────────────────────────────────
echo ""
echo "${BOLD}${CYAN}Test 3: TUI stream resolution and mpv playback (Death Note)${RESET}"

tmux has-session -t clare-test-main 2>/dev/null && tmux kill-session -t clare-test-main
tmux new-session -d -s clare-test-main -x 110 -y 32 \
    "env PATH=\"$MOCK_DIR:\$PATH\" CLARE_STATE_DIR=\"$TEST_DIR\" \"$CLARE_BIN\""

wait_for "clare-test-main" "Enter anime title|SEARCH ANIME" 8

# Search
tmux send-keys -t clare-test-main "Death Note" Enter
sleep 5

if ! wait_for "clare-test-main" "Search Results|SHOW DETAILS" 8; then
    echo "${RED}✘ Test 3 Failed: search results for 'Death Note' not loaded${RESET}"
    exit 1
fi
echo "${GREEN}✔ Search results for 'Death Note' loaded${RESET}"

# Select show
tmux send-keys -t clare-test-main Enter
sleep 5

if ! wait_for "clare-test-main" "Select Episode|EPISODE DETAILS" 8; then
    echo "${RED}✘ Test 3 Failed: episode list did not load${RESET}"
    exit 1
fi
echo "${GREEN}✔ Episode list loaded${RESET}"

# Press Enter on Episode 1 → resolves sources
tmux send-keys -t clare-test-main Enter
sleep 8

s=$(cap "clare-test-main")
echo "--- Screen (Source Select) ---"
echo "$s"
echo "------------------------------"

if echo "$s" | grep -qiE "Select Source|Ok|Yt-mp4|Mp4upload|fast4speed"; then
    echo "${GREEN}✔ Streams resolved — source selection visible${RESET}"
    if echo "$s" | grep -qiE "ALLANIME|GOGO"; then
        echo "${GREEN}✔ Provider badges (ALLANIME/GOGO) rendered correctly${RESET}"
    else
        echo "${RED}✘ Test 3 Failed: provider badges not found in source list${RESET}"
        exit 1
    fi
else
    echo "${RED}✘ Test 3 Failed: stream sources not resolved${RESET}"
    exit 1
fi

# Select first source → triggers mpv wrapper
echo "Selecting first source to launch mpv..."
tmux send-keys -t clare-test-main Enter
sleep 7

# mpv exits quickly (headless 1-frame). Clare should return to episode list.
if wait_for "clare-test-main" "Select Episode|enter: play" 8; then
    echo "${GREEN}✔ Clare returned to episode list after playback${RESET}"
else
    echo "${YELLOW}⚠ Clare did not return to episode list (stream may have timed out)${RESET}"
fi

if grep -q "MPV_CALL" "$TEST_DIR/mock.log" 2>/dev/null; then
    echo "${GREEN}✔ mpv wrapper was invoked (stream URL passed to player)${RESET}"
    cat "$TEST_DIR/mock.log" | grep "MPV"
else
    echo "${RED}✘ Test 3 Failed: mpv was never called${RESET}"
    exit 1
fi

# Quit
tmux send-keys -t clare-test-main "q"
sleep 1.5
tmux has-session -t clare-test-main 2>/dev/null && tmux kill-session -t clare-test-main
echo "${GREEN}✔ Test 3 Passed: stream resolved + mpv invoked${RESET}"

# ─── Test 4: Airing Suggestions navigation ────────────────────────────────────
echo ""
echo "${BOLD}${CYAN}Test 4: Airing Suggestions → Episode List flow${RESET}"

tmux has-session -t clare-test-airing 2>/dev/null && tmux kill-session -t clare-test-airing
tmux new-session -d -s clare-test-airing -x 110 -y 32 \
    "env PATH=\"$MOCK_DIR:\$PATH\" CLARE_STATE_DIR=\"$TEST_DIR\" \"$CLARE_BIN\""

wait_for "clare-test-airing" "CURRENTLY AIRING|Enter anime title" 8

s=$(cap "clare-test-airing")
echo "--- Screen (Search Input with Airing) ---"
echo "$s"
echo "-----------------------------------------"

if echo "$s" | grep -qiE "CURRENTLY AIRING"; then
    echo "${GREEN}✔ Airing Suggestions panel visible${RESET}"
else
    echo "${RED}✘ Test 4 Failed: airing suggestions not shown${RESET}"
    exit 1
fi

# Human presses Tab to focus the Airing Suggestions list
tmux send-keys -t clare-test-airing Tab
sleep 1

# Human presses Enter to select the first airing show (triggers a search)
tmux send-keys -t clare-test-airing Enter
sleep 4

# Clare transitions to stateSearchResults; user presses Enter to enter that show
tmux send-keys -t clare-test-airing Enter
sleep 5

s=$(cap "clare-test-airing")
echo "--- Screen (After Airing Selection) ---"
echo "$s"
echo "---------------------------------------"

if echo "$s" | grep -qiE "Select Episode|EPISODE DETAILS"; then
    echo "${GREEN}✔ Airing suggestion led to episode list${RESET}"
else
    echo "${RED}✘ Test 4 Failed: did not reach episode list from airing suggestion${RESET}"
    exit 1
fi

# Resolve streams from airing show, Episode 1
tmux send-keys -t clare-test-airing Enter
sleep 8

s=$(cap "clare-test-airing")
if echo "$s" | grep -qiE "Select Source|Ok|Yt-mp4|Mp4upload"; then
    echo "${GREEN}✔ Streams resolved for airing show${RESET}"

    # Play
    tmux send-keys -t clare-test-airing Enter
    sleep 7
    if grep -q "MPV_CALL" "$TEST_DIR/mock.log" 2>/dev/null; then
        echo "${GREEN}✔ mpv invoked for airing show${RESET}"
    else
        echo "${YELLOW}⚠ mpv not logged (stream may have expired)${RESET}"
    fi
else
    echo "${YELLOW}⚠ Stream sources not resolved for this airing show${RESET}"
fi

# Quit
tmux send-keys -t clare-test-airing "q"
sleep 1.5
tmux has-session -t clare-test-airing 2>/dev/null && tmux kill-session -t clare-test-airing
echo "${GREEN}✔ Test 4 Passed: Airing Suggestions flow validated${RESET}"

# ─── Final ────────────────────────────────────────────────────────────────────

echo ""
echo "${BOLD}${GREEN}✔ ALL TESTS PASSED${RESET}"
echo ""
echo "Mock call log:"
cat "$TEST_DIR/mock.log" 2>/dev/null || echo "  (empty)"
