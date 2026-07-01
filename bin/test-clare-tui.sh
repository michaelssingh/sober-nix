#!/usr/bin/env bash
# bin/test-clare-tui.sh — Automated TUI and CLI test script for Clare on otus.
# Runs Clare in an isolated tmux pane, sends keys, captures screens, and asserts outputs.

set -euo pipefail

BOLD=$'\e[1m'
CYAN=$'\e[36m'
GREEN=$'\e[32m'
RED=$'\e[31m'
RESET=$'\e[0m'

echo "${BOLD}${CYAN}=== Clare Automated Integration Tests ===${RESET}"

# 1. Setup isolated test state directory
TEST_DIR=$(mktemp -d -t clare-test-state.XXXXXX)
export CLARE_STATE_DIR="$TEST_DIR"
echo "Isolated state directory: $TEST_DIR"

cleanup() {
    echo -e "\n${BOLD}${CYAN}Cleaning up...${RESET}"
    rm -rf "$TEST_DIR"
    # Kill tmux session if still alive
    if tmux has-session -t clare-test-session 2>/dev/null; then
        tmux kill-session -t clare-test-session
        echo "Killed tmux test session."
    fi
}
trap cleanup EXIT

# Find clare binary
CLARE_BIN=""
echo "${BOLD}${CYAN}Building Clare locally...${RESET}"
(cd ./packages/clare && go build -o clare .)
CLARE_BIN="./packages/clare/clare"
echo "Using Clare binary: $CLARE_BIN"

# --- Test 1: Non-interactive Version Flag ---
echo -e "\n${BOLD}${CYAN}Running Test 1: CLI Version Flag...${RESET}"
version_out=$("$CLARE_BIN" -version)
echo "Output: $version_out"
if [[ "$version_out" =~ ^clare\ [0-9]+\.[0-9]+\.[0-9]+ ]]; then
    echo "${GREEN}✔ Test 1 Passed: Version flag works.${RESET}"
else
    echo "${RED}✘ Test 1 Failed: Version format mismatch.${RESET}"
    exit 1
fi

# --- Test 2: Non-interactive Direct Mode (Dry Run/Timeout) ---
echo -e "\n${BOLD}${CYAN}Running Test 2: CLI Direct Download (Death Note Ep 1)...${RESET}"
# We start the command in the background, wait 7 seconds to see if it starts downloading/resolving, then kill it.
# This validates search API -> episode selection -> stream resolution -> launch downloader flow.
echo "Starting direct download in background..."
"$CLARE_BIN" -s "Death Note" -e "1" -d > "$TEST_DIR/cli_download.log" 2>&1 &
CLI_PID=$!

sleep 7

if kill -0 "$CLI_PID" 2>/dev/null; then
    echo "Process is running (active download/resolution). Killing it to pass test."
    kill "$CLI_PID"
else
    # Process ended. Let's clean up/wait
    wait "$CLI_PID" || true
fi

cat "$TEST_DIR/cli_download.log"

# Verify log output contains key phrases showing it got to stream resolution
if grep -q -E "Resolving stream|Download completed" "$TEST_DIR/cli_download.log"; then
    echo "${GREEN}✔ Test 2 Passed: CLI direct mode successfully triggered stream resolution.${RESET}"
else
    echo "${RED}✘ Test 2 Failed: Log output missing expected CLI direct flow progress lines.${RESET}"
    exit 1
fi

# --- Test 3: Interactive TUI (via tmux) ---
echo -e "\n${BOLD}${CYAN}Running Test 3: Interactive TUI flow (tmux)...${RESET}"

# Ensure no existing session conflicts
if tmux has-session -t clare-test-session 2>/dev/null; then
    tmux kill-session -t clare-test-session
fi

# Create a tmux session with standard layout (100 columns to fit details panels)
tmux new-session -d -s clare-test-session -x 100 -y 30 "env CLARE_STATE_DIR=\"$TEST_DIR\" \"$CLARE_BIN\""

echo "Waiting for Clare TUI to start..."
sleep 2

# Capture initial screen
tui_screen_1=$(tmux capture-pane -p -t clare-test-session)
echo "--- Screen Capture 1 (Initial Search Input) ---"
echo "$tui_screen_1"
echo "-----------------------------------------------"

# Assert we are in the search input state
if echo "$tui_screen_1" | grep -q -E "Search Anime|Enter anime title"; then
    echo "${GREEN}✔ Initial TUI state: Search Input screen confirmed.${RESET}"
else
    echo "${RED}✘ Test 3 Failed: TUI did not start in search input state.${RESET}"
    exit 1
fi

# Send search query "Frieren"
echo "Sending search query 'Frieren' to TUI..."
tmux send-keys -t clare-test-session "Frieren" Enter
echo "Waiting for search results and cover art details..."
sleep 5

# Capture search results screen
tui_screen_2=$(tmux capture-pane -p -t clare-test-session)
echo "--- Screen Capture 2 (Search Results & Cover Art) ---"
echo "$tui_screen_2"
echo "-----------------------------------------------------"

# Assert we show Search Results, Frieren, and Details Panel
if echo "$tui_screen_2" | grep -q -E "Search Results|Frieren" && echo "$tui_screen_2" | grep -q "SHOW DETAILS"; then
    echo "${GREEN}✔ TUI Search Results: List and Details panels rendered correctly.${RESET}"
else
    echo "${RED}✘ Test 3 Failed: TUI did not display search results or details panel.${RESET}"
    exit 1
fi

# Press Enter to select the show and fetch episodes list
echo "Selecting highlighted show..."
tmux send-keys -t clare-test-session Enter
echo "Waiting for episode list to fetch..."
sleep 5

# Capture episode list screen
tui_screen_3=$(tmux capture-pane -p -t clare-test-session)
echo "--- Screen Capture 3 (Episode Select) ---"
echo "$tui_screen_3"
echo "-----------------------------------------"

if echo "$tui_screen_3" | grep -q "Select Episode" && echo "$tui_screen_3" | grep -q -E "Episode 1|Ep 1"; then
    echo "${GREEN}✔ TUI Episode Select: Episode list populated and displayed.${RESET}"
else
    echo "${RED}✘ Test 3 Failed: TUI did not load or display episode selection.${RESET}"
    exit 1
fi

# Test Esc key to return to Search Results
echo "Testing ESC key back to show list..."
tmux send-keys -t clare-test-session Escape
sleep 2

tui_screen_4=$(tmux capture-pane -p -t clare-test-session)
if echo "$tui_screen_4" | grep -q "Search Results" && echo "$tui_screen_4" | grep -q "SHOW DETAILS"; then
    echo "${GREEN}✔ TUI Back Navigation: Successfully returned to show selection.${RESET}"
else
    echo "${RED}✘ Test 3 Failed: ESC key did not return to show list.${RESET}"
    exit 1
fi

# Quit Clare TUI gracefully
echo "Quitting Clare TUI..."
tmux send-keys -t clare-test-session "q"
sleep 1.5

if tmux has-session -t clare-test-session 2>/dev/null; then
    echo "Clare did not exit on 'q'. Killing session..."
    tmux kill-session -t clare-test-session
    echo "${RED}✘ Test 3 Failed: Clare TUI did not exit gracefully on 'q'.${RESET}"
    exit 1
else
    echo "${GREEN}✔ Test 3 Passed: Interactive TUI flow and keybinds validated successfully!${RESET}"
fi

echo -e "\n${BOLD}${GREEN}✔ ALL TESTS PASSED SUCCESSFULLY!${RESET}"
