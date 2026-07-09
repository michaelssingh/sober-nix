package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tuiState int

const (
	stateHistory tuiState = iota
	stateSearchInput
	stateSearchRunning
	stateShowSelect
	stateEpisodeSelect
	stateSourceSelect
	statePlaybackPreparing
	statePlaybackActive
	stateError
	stateLogs
	stateConfig
)

// Styles for premium aesthetic
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#1f2335")). // dark text on accent background
			Background(lipgloss.Color("#7aa2f7")). // Tokyonight blue
			Padding(0, 2).
			MarginBottom(1)

	accentColorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#bb9af7")) // Tokyonight magenta

	cyanColorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7dcfff")) // Tokyonight cyan

	grayColorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89")) // Tokyonight comment/gray

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f7768e")) // Tokyonight red

	normalTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#c0caf5")) // Tokyonight foreground

	selectedTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#bb9af7")) // Tokyonight magenta

	// Sub/Dub translation badges
	subBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9ece6a")) // Tokyonight green

	dubBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7dcfff")) // Tokyonight cyan/blue

	subDubBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#bb9af7")) // Tokyonight magenta/purple
)

// List items definitions

type historyItem struct {
	showID      string
	showName    string
	lastEp      string
	timestamp   int64
	isCompleted bool
	totalEps    int     // total episodes (from positions or cache), 0 = unknown
	nextEp      string  // next episode to watch, empty if none/completed
	progressPct float64 // 0.0-1.0 progress, -1 if unknown
}

func (h historyItem) Title() string {
	if h.isCompleted {
		return "✓ " + h.showName
	}
	return h.showName
}

func (h historyItem) Description() string {
	ago := humanAgo(h.timestamp)
	if h.isCompleted {
		if h.totalEps > 0 {
			return fmt.Sprintf("Completed — %d episodes  ·  %s", h.totalEps, ago)
		}
		return fmt.Sprintf("Completed  ·  %s", ago)
	}
	var parts []string
	if h.nextEp != "" && h.totalEps > 0 {
		parts = append(parts, fmt.Sprintf("Next: Ep %s / %d", h.nextEp, h.totalEps))
	} else if h.nextEp != "" {
		parts = append(parts, fmt.Sprintf("Next: Ep %s", h.nextEp))
	} else if h.lastEp != "" {
		parts = append(parts, fmt.Sprintf("Last: Ep %s", h.lastEp))
	}
	if h.progressPct >= 0 && h.totalEps > 0 {
		bar := renderSmoothProgressBar(h.progressPct, 8)
		parts = append(parts, fmt.Sprintf("[%s] %d%%", bar, int(h.progressPct*100)))
	}
	parts = append(parts, ago)
	return strings.Join(parts, "  ·  ")
}

func renderSmoothProgressBar(pct float64, width int) string {
	if pct < 0 {
		return ""
	}
	if pct > 1.0 {
		pct = 1.0
	}

	blocks := []string{" ", "▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}
	fullBlocksCount := int(pct * float64(width))
	remainder := pct*float64(width) - float64(fullBlocksCount)
	remainderIndex := int(remainder * 8)

	var bar strings.Builder
	for i := 0; i < fullBlocksCount; i++ {
		bar.WriteString("█")
	}
	if fullBlocksCount < width {
		bar.WriteString(blocks[remainderIndex])
		for i := fullBlocksCount + 1; i < width; i++ {
			bar.WriteString(" ")
		}
	}
	
	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff")) // Tokyonight ice blue
	return barStyle.Render(bar.String())
}

func (h historyItem) FilterValue() string { return h.showName }

type showItem struct {
	show AnimeShow
}

func (s showItem) Title() string       { return s.show.Name }
func (s showItem) Description() string {
	var parts []string
	if s.show.Type != "" {
		parts = append(parts, s.show.Type)
	}
	if s.show.Season.Quarter != "" && s.show.Season.Year > 0 {
		parts = append(parts, fmt.Sprintf("%s %d", s.show.Season.Quarter, s.show.Season.Year))
	}
	if s.show.Score > 0 {
		parts = append(parts, fmt.Sprintf("Score: %.2f", s.show.Score))
	}
	parts = append(parts, fmt.Sprintf("%d episodes available", s.show.EpCount()))

	// Add sub/dub badge
	subCount := s.show.SubCount()
	dubCount := s.show.DubCount()
	if subCount > 0 && dubCount > 0 {
		parts = append(parts, subDubBadgeStyle.Render("SUB+DUB"))
	} else if subCount > 0 {
		parts = append(parts, subBadgeStyle.Render("SUB"))
	} else if dubCount > 0 {
		parts = append(parts, dubBadgeStyle.Render("DUB"))
	}

	return strings.Join(parts, "  •  ")
}
func (s showItem) FilterValue() string { return s.show.Name }

type episodeItem struct {
	epNo        string
	isNext      bool
	isCompleted bool
	title       string
	desc        string
	subAvail    bool
	dubAvail    bool
}

func (e episodeItem) Title() string {
	base := ""
	if e.title != "" {
		base = e.title
	} else {
		base = fmt.Sprintf("Episode %s", e.epNo)
	}
	if e.isCompleted {
		base = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")).Render("✔ ") + base
	}
	if e.isNext {
		base = fmt.Sprintf("%s (Next Up)", base)
	}

	// Add sub/dub badges
	var badge string
	if e.subAvail && e.dubAvail {
		badge = " " + subDubBadgeStyle.Render("[SUB+DUB]")
	} else if e.subAvail {
		badge = " " + subBadgeStyle.Render("[SUB]")
	} else if e.dubAvail {
		badge = " " + dubBadgeStyle.Render("[DUB]")
	}

	return base + badge
}
func (e episodeItem) Description() string { return e.desc }
func (e episodeItem) FilterValue() string { return e.epNo }

type JikanEpInfo struct {
	Title    string
	Aired    string
	Synopsis string
	Filler   bool
	Recap    bool
}

type jikanMetadataMsg struct {
	malID    string
	page     int
	metadata map[string]JikanEpInfo
	err      error
}

// Msg definitions
type searchResultMsg struct {
	shows []AnimeShow
	err   error
}

type episodesResultMsg struct {
	show     AnimeShow
	episodes []string
	err      error
}

type showDetailsResultMsg struct {
	showID string
	show   AnimeShow
	err    error
}

type allStreamsResultMsg struct {
	epNo    string
	streams []ResolvedStream
	err     error
}

type tickMpvStatusMsg struct {
	status MpvStatus
	err    error
}

type sourceItem struct {
	stream ResolvedStream
}

func (s sourceItem) Title() string {
	return fmt.Sprintf("%s (%s)", s.stream.SourceName, s.stream.Quality)
}
func (s sourceItem) Description() string {
	return s.stream.URL
}
func (s sourceItem) FilterValue() string {
	return s.stream.SourceName
}

type CoverArtLoadedMsg struct {
	ShowID string
	Ansi   string
}


type aniSkipCheckedMsg struct {
	epNo  string
	ready bool
}

type resolvedPlaybackMsg struct {
	warning          string
	cmd              *exec.Cmd
	tempLuaFile      string
	tempChaptersFile string
	err              error
}

type playbackFinishedMsg struct {
	err              error
	tempLuaFile      string
	tempChaptersFile string
}

// Bubble Tea Model
type model struct {
	state               tuiState
	historyItems        []list.Item
	historyList         list.Model
	searchInput         textinput.Model
	spinner             spinner.Model
	showItems           []list.Item
	showList            list.Model
	episodeItems        []list.Item
	episodeList         list.Model
	selectedShow        AnimeShow
	selectedEp          string
	playingShow         AnimeShow
	playingEp           string
	episodes            []string
	download            bool
	quality             string
	mode                string // sub, dub, dual
	err                 error
	width, height       int
	loadingMsg          string
	tempLuaFile         string
	tempChaptersFile    string
	initialSearch       string
	episodeDetails      map[string]JikanEpInfo
	loadedJikanPages    map[int]bool
	fetchingSynopsis    map[string]bool
	autoplay            bool
	autoskip            bool
	skipFillers         bool
	configCursor        int
	triggerAutoplay     bool
	historyShowDetails  map[string]AnimeShow
	coverArtCache       map[string]string
	detailsScrollOffset int
	lastSelectedShowID  string
	telemetryLogs       []string
	telemetryViewport   viewport.Model
	showTelemetry       bool
	aniSkipReady        map[string]bool
	activeCmd           *exec.Cmd
	clareLogChan        chan string
	showCompleted       bool // whether to include completed shows in history list
	searchHistory       []string
	searchHistoryIndex  int
	sourceList          list.Model
	resolvedStreams     []ResolvedStream
	mpvStatus           MpvStatus
	playbackActive      bool
}

func createMinimalList(title string) list.Model {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#bb9af7")).
		BorderLeftForeground(lipgloss.Color("#bb9af7"))
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#565f89")).
		BorderLeftForeground(lipgloss.Color("#bb9af7"))
	d.Styles.NormalTitle = d.Styles.NormalTitle.
		Foreground(lipgloss.Color("#c0caf5"))
	d.Styles.NormalDesc = d.Styles.NormalDesc.
		Foreground(lipgloss.Color("#565f89"))

	l := list.New([]list.Item{}, d, 0, 0)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#c0caf5")).
		Background(lipgloss.Color("#1f2335")).
		Padding(0, 1)

	return l
}

// createHistoryList creates the history list with a custom delegate that dims completed items.
func createHistoryList() list.Model {
	d := list.NewDefaultDelegate()
	// Active (in-progress) items — vivid
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#bb9af7")).
		BorderLeftForeground(lipgloss.Color("#bb9af7"))
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#565f89")).
		BorderLeftForeground(lipgloss.Color("#bb9af7"))
	d.Styles.NormalTitle = d.Styles.NormalTitle.
		Foreground(lipgloss.Color("#c0caf5"))
	d.Styles.NormalDesc = d.Styles.NormalDesc.
		Foreground(lipgloss.Color("#565f89"))

	l := list.New([]list.Item{}, d, 0, 0)
	l.Title = "Continue Watching"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7aa2f7")).
		Background(lipgloss.Color("#1f2335")).
		Padding(0, 1)

	return l
}

func initialModel(initialSearch, mode, quality string, download bool) model {
	// Setup spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#bb9af7"))

	// Setup searchInput
	ti := textinput.New()
	ti.Placeholder = "Enter anime title..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 30
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#c0caf5"))

	hList := createHistoryList()
	sList := createMinimalList("Search Results")
	eList := createMinimalList("Select Episode")
	soList := createMinimalList("Select Source & Resolution")

	cfg := loadConfig()

	// Reattach to running MPV if active
	var isReattached bool
	var activeShow AnimeShow
	var activeEp string
	var reattachedStatus MpvStatus

	status, err := queryMpvStatus()
	if err == nil {
		titleVal, errTitle := queryMediaTitle()
		if errTitle == nil && titleVal != "" {
			parts := strings.Split(titleVal, " - Episode ")
			if len(parts) == 2 {
				showName := parts[0]
				epNo := parts[1]
				
				var foundShow AnimeShow
				var found bool
				hist, _ := loadHistory()
				for _, h := range hist {
					if h.ShowName == showName {
						foundShow, _, found = loadShowCache(h.ShowID)
						break
					}
				}
				if !found {
					foundShow = AnimeShow{Name: showName, EnglishName: showName}
				}
				activeShow = foundShow
				activeEp = epNo
				reattachedStatus = status
				isReattached = true
				debugLog("[INFO] Reattached to running MPV: %s (Ep %s)", showName, epNo)
			}
		}
	} else {
		if _, statErr := os.Stat("/tmp/clare-mpv.sock"); statErr == nil {
			debugLog("[INFO] Removing stale MPV socket file...")
			_ = os.Remove("/tmp/clare-mpv.sock")
		}
	}

	m := model{
		state:              stateHistory,
		historyList:        hList,
		searchInput:        ti,
		spinner:            s,
		showList:           sList,
		episodeList:        eList,
		sourceList:         soList,
		mode:               mode,
		quality:            quality,
		download:           download,
		initialSearch:      initialSearch,
		episodeDetails:     make(map[string]JikanEpInfo),
		loadedJikanPages:   make(map[int]bool),
		fetchingSynopsis:   make(map[string]bool),
		autoplay:           cfg.Autoplay,
		autoskip:           cfg.Autoskip,
		skipFillers:        cfg.SkipFillers,
		historyShowDetails: make(map[string]AnimeShow),
		coverArtCache:      make(map[string]string),
		telemetryViewport:  viewport.New(0, 0),
		showTelemetry:      true, // Enabled by default
		aniSkipReady:       make(map[string]bool),
		clareLogChan:       make(chan string, 1000),
	}

	if isReattached {
		m.playbackActive = true
		m.selectedShow = activeShow
		m.selectedEp = activeEp
		m.playingShow = activeShow
		m.playingEp = activeEp
		m.mpvStatus = reattachedStatus
	}

	m.refreshHistory()
	go tailLogFile(m.clareLogChan)

	if initialSearch != "" {
		m.state = stateSearchRunning
		m.loadingMsg = fmt.Sprintf("Searching for %q...", initialSearch)
	} else if len(m.historyItems) == 0 {
		m.state = stateSearchInput
		m.searchHistory, _ = loadSearchHistory()
		m.searchHistoryIndex = -1
	}

	return m
}

func (m *model) enterSearchState() {
	m.state = stateSearchInput
	m.searchInput.Reset()
	m.searchInput.Focus()
	m.searchHistory, _ = loadSearchHistory()
	m.searchHistoryIndex = -1
}


func (m *model) refreshHistory() {
	rawHist, err := loadHistory()
	if err != nil {
		return
	}
	uniq := getUniqueHistory(rawHist)
	positions, _ := loadPositions()

	var allItems []list.Item
	for _, u := range uniq {
		item := historyItem{
			showID:      u.ShowID,
			showName:    u.ShowName,
			lastEp:      u.Episode,
			timestamp:   u.Timestamp,
			progressPct: -1,
		}

		// Enrich from positions.json if available. We need the MALID — look it up
		// from the show cache if available.
		var malID string
		if cached, _, found := loadShowCache(u.ShowID); found {
			malID = cached.MALID
			item.totalEps = cached.EpCount()
		}

		if positions != nil && malID != "" && malID != "0" {
			if showState, ok := positions[malID]; ok {
				// Determine if completed
				maxCompleted := 0.0
				for _, ep := range showState.CompletedEpisodes {
					if ep > maxCompleted {
						maxCompleted = ep
					}
				}
				lastEpVal := parseEpisodeNumber(item.lastEp)
				completed := false
				if item.totalEps > 0 {
					completed = (maxCompleted >= float64(item.totalEps)) ||
						(lastEpVal >= float64(item.totalEps)) ||
						(len(showState.CompletedEpisodes) >= item.totalEps)
				}
				item.isCompleted = completed

				if !completed {
					if showState.ResumeState != nil {
						// In-progress episode
						nextVal := maxCompleted + 1.0
						if showState.ResumeState.Episode > maxCompleted {
							nextVal = showState.ResumeState.Episode
						}
						epStr := fmt.Sprintf("%.1f", nextVal)
						if strings.HasSuffix(epStr, ".0") {
							epStr = epStr[:len(epStr)-2]
						}
						item.nextEp = epStr
						// Compute progress based on maxCompleted
						maxEp := 0.0
						for _, cEp := range showState.CompletedEpisodes {
							if cEp > maxEp {
								maxEp = cEp
							}
						}
						currEpVal := parseEpisodeNumber(epStr)
						if currEpVal > maxEp {
							maxEp = currEpVal
						}
						if item.totalEps > 0 {
							item.progressPct = maxEp / float64(item.totalEps)
						}
					} else if u.Episode != "" {
						// Finished an episode, next sequential one up
						nextVal := maxCompleted + 1.0
						epVal := parseEpisodeNumber(u.Episode)
						if epVal > maxCompleted {
							nextVal = epVal + 1.0
						}
						if item.totalEps > 0 {
							if int(nextVal) <= item.totalEps {
								if nextVal == float64(int(nextVal)) {
									item.nextEp = fmt.Sprintf("%d", int(nextVal))
								} else {
									item.nextEp = fmt.Sprintf("%.1f", nextVal)
								}
							}
							// Progress bar percentage uses max completed episode
							maxEp := 0.0
							for _, cEp := range showState.CompletedEpisodes {
								if cEp > maxEp {
									maxEp = cEp
								}
							}
							item.progressPct = maxEp / float64(item.totalEps)
						} else {
							item.nextEp = fmt.Sprintf("%.0f", nextVal)
						}
					}
				}
			}
		} else {
			// No positions data — fall back to history to set next ep heuristic
			if u.Episode != "" && item.totalEps > 0 {
				epVal := parseEpisodeNumber(u.Episode)
				nextVal := epVal + 1
				if int(nextVal) <= item.totalEps {
					item.nextEp = fmt.Sprintf("%.0f", nextVal)
					item.progressPct = epVal / float64(item.totalEps)
				} else if int(epVal) >= item.totalEps {
					item.isCompleted = true
				}
			}
		}

		allItems = append(allItems, item)
	}
	m.historyItems = allItems
	m.applyHistoryFilter()
}

// applyHistoryFilter updates the list model based on the showCompleted toggle.
func (m *model) applyHistoryFilter() {
	var filtered []list.Item
	var completedCount, activeCount int
	for _, it := range m.historyItems {
		if h, ok := it.(historyItem); ok {
			if h.isCompleted {
				completedCount++
				if m.showCompleted {
					filtered = append(filtered, it)
				}
			} else {
				activeCount++
				filtered = append(filtered, it)
			}
		}
	}
	// Update list title with counts
	if m.showCompleted {
		m.historyList.Title = fmt.Sprintf("Watching (%d)  +  Completed (%d)", activeCount, completedCount)
	} else if completedCount > 0 {
		m.historyList.Title = fmt.Sprintf("Continue Watching (%d)  [c: show %d completed]", activeCount, completedCount)
	} else {
		m.historyList.Title = fmt.Sprintf("Continue Watching (%d)", activeCount)
	}
	m.historyList.SetItems(filtered)
}

func (m *model) toggleShowCompleted(showID string) {
	if showID == "" {
		return
	}
	var malID string
	var totalEps int
	if cached, _, found := loadShowCache(showID); found {
		malID = cached.MALID
		totalEps = cached.EpCount()
	}
	if malID == "" || malID == "0" {
		debugLog("toggleShowCompleted: show %s has no valid MAL ID, cannot toggle", showID)
		return
	}

	positions, err := loadPositions()
	if err != nil {
		positions = make(map[string]ShowState)
	}

	showState, ok := positions[malID]
	if !ok {
		showState = ShowState{
			CompletedEpisodes: []float64{},
		}
	}

	// Determine if completed
	maxCompleted := 0.0
	for _, ep := range showState.CompletedEpisodes {
		if ep > maxCompleted {
			maxCompleted = ep
		}
	}

	isCompleted := false
	if totalEps > 0 {
		isCompleted = (maxCompleted >= float64(totalEps)) || (len(showState.CompletedEpisodes) >= totalEps)
	} else if len(showState.CompletedEpisodes) > 0 {
		isCompleted = true
	}

	if isCompleted {
		// Currently completed: mark it as in-progress by clearing the max episode or everything from CompletedEpisodes
		showState.CompletedEpisodes = []float64{}
		showState.ResumeState = nil
		debugLog("toggleShowCompleted: unmarking show %s (MAL %s) as completed", showID, malID)
	} else {
		// Mark it completed: clear ResumeState and add totalEps (or a high number like 999) to CompletedEpisodes
		targetEp := 999.0
		if totalEps > 0 {
			targetEp = float64(totalEps)
		}
		// Ensure it's not already in list
		found := false
		for _, ep := range showState.CompletedEpisodes {
			if ep == targetEp {
				found = true
				break
			}
		}
		if !found {
			showState.CompletedEpisodes = append(showState.CompletedEpisodes, targetEp)
		}
		showState.ResumeState = nil
		debugLog("toggleShowCompleted: marking show %s (MAL %s) as completed", showID, malID)
	}

	positions[malID] = showState
	_ = savePositions(positions)
	m.refreshHistory()
}


func humanAgo(ts int64) string {
	if ts == 0 {
		return "unknown"
	}
	dur := time.Since(time.Unix(ts, 0))
	switch {
	case dur < time.Minute:
		return "just now"
	case dur < time.Hour:
		return fmt.Sprintf("%dm ago", int(dur.Minutes()))
	case dur < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(dur.Hours()))
	case dur < 7*24*time.Hour:
		days := int(dur.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	case dur < 30*24*time.Hour:
		weeks := int(dur.Hours() / (24 * 7))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		return time.Unix(ts, 0).Format("2006-01-02")
	}
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, m.spinner.Tick)
	cmds = append(cmds, readClareLogsCmd(m.clareLogChan))
	if m.playbackActive {
		cmds = append(cmds, tickMpvStatusCmd())
	}
	if m.initialSearch != "" {
		cmds = append(cmds, doSearch(m.initialSearch, "sub"))
	} else if m.state == stateHistory {
		if len(m.historyItems) > 0 {
			if selected, ok := m.historyItems[0].(historyItem); ok {
				if show, _, found := loadShowCache(selected.showID); found {
					m.historyShowDetails[selected.showID] = show
					if show.Thumbnail != "" {
						m.coverArtCache[selected.showID] = "Loading..."
						cmds = append(cmds, doFetchCoverArt(selected.showID, show.Thumbnail, 16, 11))
					}
				} else {
					m.historyShowDetails[selected.showID] = AnimeShow{
						ID:          selected.showID,
						Name:        selected.showName,
						Description: "Loading details...",
					}
					cmds = append(cmds, doFetchShowDetails(selected.showID))
				}
			}
		}
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalculateSizes()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case searchResultMsg:
		debugLog("TUI searchResultMsg: shows=%d, err=%v", len(msg.shows), msg.err)
		m.state = stateShowSelect
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		if len(msg.shows) == 0 {
			m.state = stateError
			m.err = fmt.Errorf("no shows found matching query")
			return m, nil
		}

		var items []list.Item
		for _, s := range msg.shows {
			items = append(items, showItem{show: s})
		}
		m.showItems = items
		m.showList.SetItems(items)
		
		// If there is only one show, auto-select it and fetch episodes immediately
		if len(msg.shows) == 1 {
			m.selectedShow = msg.shows[0]
			m.state = stateSearchRunning
			m.loadingMsg = fmt.Sprintf("Fetching episode list for %s...", m.selectedShow.Name)
			return m, doFetchEpisodes(m.selectedShow.ID, "sub")
		}

		if len(msg.shows) > 1 {
			firstShow := msg.shows[0]
			if firstShow.Thumbnail != "" {
				if _, ok := m.coverArtCache[firstShow.ID]; !ok {
					m.coverArtCache[firstShow.ID] = "Loading..."
					return m, doFetchCoverArt(firstShow.ID, firstShow.Thumbnail, 16, 11)
				}
			}
		}
		return m, nil

	case episodesResultMsg:
		debugLog("TUI episodesResultMsg: episodes=%d, err=%v", len(msg.episodes), msg.err)
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}

		m.selectedShow = msg.show
		m.episodes = msg.episodes
		m.state = stateEpisodeSelect

		// Clear metadata maps
		m.episodeDetails = make(map[string]JikanEpInfo)
		m.loadedJikanPages = make(map[int]bool)
		m.fetchingSynopsis = make(map[string]bool)

		// Load Jikan cache if MAL ID is present
		if m.selectedShow.MALID != "" && m.selectedShow.MALID != "0" {
			if cacheData, err := loadJikanCache(m.selectedShow.MALID); err == nil && len(cacheData) > 0 {
				for k, v := range cacheData {
					m.episodeDetails[k] = v
				}
				// Pre-mark pages that are fully cached
				for _, ep := range m.episodes {
					var val int
					fmt.Sscanf(ep, "%d", &val)
					if val > 0 {
						page := (val - 1) / 100 + 1
						if _, ok := cacheData[ep]; ok {
							m.loadedJikanPages[page] = true
						}
					}
				}
			}
		}

		// Sort episodes in ascending order
		sort.Slice(m.episodes, func(i, j int) bool {
			valI := parseEpisodeNumber(m.episodes[i])
			valJ := parseEpisodeNumber(m.episodes[j])
			if valI == valJ {
				return m.episodes[i] < m.episodes[j]
			}
			return valI < valJ
		})

		// Rebuild the list items with current (empty/fallback) titles
		m.refreshEpisodeListItems()

		// Set list selection to the next up episode (or index 0 if not found)
		selectIndex := 0
		for i, item := range m.episodeItems {
			if epItem, ok := item.(episodeItem); ok && epItem.isNext {
				selectIndex = i
				break
			}
		}
		m.episodeList.Select(selectIndex)

		if selectIndex < len(m.episodeItems) {
			if epItem, ok := m.episodeItems[selectIndex].(episodeItem); ok {
				prefetchEpisodeStream(m.selectedShow.ID, m.mode, epItem.epNo, m.quality)
			}
			if selectIndex+1 < len(m.episodeItems) {
				if nextEpItem, ok := m.episodeItems[selectIndex+1].(episodeItem); ok {
					prefetchEpisodeStream(m.selectedShow.ID, m.mode, nextEpItem.epNo, m.quality)
				}
			}
		}

		// If autoplay was triggered, fetch next stream immediately
		if m.triggerAutoplay {
			m.triggerAutoplay = false
			currEpVal := parseEpisodeNumber(m.playingEp)
			var nextEpNo string
			foundNext := false
			
			totalEps := m.selectedShow.EpCount()
			if totalEps == 0 {
				totalEps = 1000
			}
			
			if currEpVal > 0 && int(currEpVal) < totalEps {
				nextEpVal := currEpVal + 1.0
				if nextEpVal == float64(int(nextEpVal)) {
					nextEpNo = fmt.Sprintf("%d", int(nextEpVal))
				} else {
					nextEpNo = fmt.Sprintf("%.1f", nextEpVal)
				}
				foundNext = true
			}
			
			if foundNext && nextEpNo != "" && m.skipFillers {
				// Cycle checking if next episode is a filler
				for {
					if info, ok := m.episodeDetails[nextEpNo]; ok && info.Filler {
						debugLog("[INFO] Autoplay: Skipping filler episode %s", nextEpNo)
						currVal := parseEpisodeNumber(nextEpNo)
						nextVal := currVal + 1.0
						if int(currVal) < totalEps {
							if nextVal == float64(int(nextVal)) {
								nextEpNo = fmt.Sprintf("%d", int(nextVal))
							} else {
								nextEpNo = fmt.Sprintf("%.1f", nextVal)
							}
						} else {
							nextEpNo = ""
							break
						}
					} else {
						break
					}
				}
			}

			if foundNext && nextEpNo != "" {
				m.selectedEp = nextEpNo
				m.state = statePlaybackPreparing
				m.loadingMsg = fmt.Sprintf("Autoplay: Preparing playback for Episode %s...", nextEpNo)
				return m, doPreparePlayback(m.selectedShow, nextEpNo, m.mode, m.quality, m.download)
			}
		}

		// Determine which page to fetch first
		lastEp := ""
		rawHist, _ := loadHistory()
		for _, h := range rawHist {
			if h.ShowID == m.selectedShow.ID {
				lastEp = h.Episode
				break
			}
		}

		page := 1
		if lastEp != "" {
			var val int
			fmt.Sscanf(lastEp, "%d", &val)
			if val > 0 {
				page = (val - 1) / 100 + 1
			}
		}
		m.loadedJikanPages[page] = true

		return m, doFetchJikanMetadata(m.selectedShow.MALID, page)

	case resolvedPlaybackMsg:
		debugLog("[INFO] resolvedPlaybackMsg: err=%v, warning=%s, tempLuaFile=%s, tempChaptersFile=%s", msg.err, msg.warning, msg.tempLuaFile, msg.tempChaptersFile)
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		if msg.warning != "" {
			m.loadingMsg = msg.warning
		}

		// Try to reuse the running MPV instance
		reused := false
		if m.playbackActive {
			args := msg.cmd.Args
			if len(args) >= 2 {
				streamURL := args[len(args)-1]
				var extraArgs []string
				for _, arg := range args[1 : len(args)-1] {
					if strings.HasPrefix(arg, "--audio-file=") {
						extraArgs = append(extraArgs, arg)
					}
				}
				
				debugLog("[INFO] Attempting to load next file in existing MPV via IPC...")
				err := loadFileInMpv(streamURL, m.selectedShow.Name, m.selectedEp, m.selectedShow.MALID, extraArgs)
				if err == nil {
					debugLog("[INFO] Successfully loaded next episode via IPC!")
					reused = true
					
					// Clean up the old temp files
					if m.tempLuaFile != "" {
						_ = os.Remove(m.tempLuaFile)
					}
					if m.tempChaptersFile != "" {
						_ = os.Remove(m.tempChaptersFile)
					}
					
					m.tempLuaFile = msg.tempLuaFile
					m.tempChaptersFile = msg.tempChaptersFile
					
					// Dynamically set new chapters-file in MPV
					if msg.tempChaptersFile != "" {
						conn, errIPC := net.DialTimeout("unix", "/tmp/clare-mpv.sock", 100*time.Millisecond)
						if errIPC == nil {
							_, _ = sendMpvCommand(conn, []interface{}{"set_property", "chapters-file", msg.tempChaptersFile})
							conn.Close()
						}
					}
				} else {
					debugLog("[WARN] IPC loadfile failed: %v. Starting new MPV process...", err)
				}
			}
		}

		if !reused {
			if m.activeCmd != nil {
				debugLog("[INFO] resolvedPlaybackMsg: stopping existing playback process")
				_ = m.activeCmd.Process.Kill()
				_ = m.activeCmd.Wait()
				if m.tempLuaFile != "" {
					_ = os.Remove(m.tempLuaFile)
				}
				if m.tempChaptersFile != "" {
					_ = os.Remove(m.tempChaptersFile)
				}
			}
			m.activeCmd = msg.cmd
			m.tempLuaFile = msg.tempLuaFile
			m.tempChaptersFile = msg.tempChaptersFile

			stdout, err := msg.cmd.StdoutPipe()
			if err != nil {
				m.state = stateError
				m.err = err
				return m, nil
			}
			stderr, err := msg.cmd.StderrPipe()
			if err != nil {
				m.state = stateError
				m.err = err
				return m, nil
			}

			if err := msg.cmd.Start(); err != nil {
				m.state = stateError
				m.err = err
				return m, nil
			}

			debugLog("[INFO] --- Playback Started: %s (Ep %s) ---", m.selectedShow.Name, m.selectedEp)

			go func() {
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					debugLog("[MPV] %s", scanner.Text())
				}
			}()

			go func() {
				scanner := bufio.NewScanner(stderr)
				for scanner.Scan() {
					debugLog("[MPV] %s", scanner.Text())
				}
			}()
		}

		m.playbackActive = true
		m.playingShow = m.selectedShow
		m.playingEp = m.selectedEp
		m.mpvStatus = MpvStatus{Paused: false, Volume: 100}

		if m.selectedShow.ID != "" {
			m.state = stateEpisodeSelect
		} else {
			m.state = stateHistory
		}

		var exitCmd tea.Cmd
		if !reused {
			exitCmd = waitForExitCmd(msg.cmd, msg.tempLuaFile, msg.tempChaptersFile)
		}

		return m, tea.Batch(
			exitCmd,
			tickMpvStatusCmd(),
		)

	case tickMpvStatusMsg:
		if m.playbackActive {
			if msg.err == nil {
				m.mpvStatus = msg.status
				
				// Autoplay trigger: if playback time reaches near duration
				if m.mpvStatus.Duration > 0 && m.mpvStatus.PlaybackTime >= m.mpvStatus.Duration - 1.5 {
					if m.autoplay && !m.triggerAutoplay {
						m.triggerAutoplay = true
						// Save progress history for current episode
						_ = recordWatch(m.selectedShow.ID, m.selectedShow.Name, m.selectedEp)
						m.refreshHistory()
					}
				}
			}
			return m, tickMpvStatusCmd()
		}
		return m, nil

	case syncRefreshMsg:
		m.refreshHistory()
		return m, nil

	case clareLogMsg:
		line := string(msg)
		if strings.Contains(line, "TUI KeyMsg:") {
			return m, readClareLogsCmd(m.clareLogChan)
		}
		wasAtBottom := m.telemetryViewport.AtBottom()
		
		isMpvProgress := func(l string) bool {
			return strings.Contains(l, "[MPV] AV:") || strings.Contains(l, "[MPV] (Paused) AV:")
		}

		if len(m.telemetryLogs) > 0 && isMpvProgress(line) && isMpvProgress(m.telemetryLogs[len(m.telemetryLogs)-1]) {
			m.telemetryLogs[len(m.telemetryLogs)-1] = line
		} else {
			m.telemetryLogs = append(m.telemetryLogs, line)
			if len(m.telemetryLogs) > 1000 {
				m.telemetryLogs = m.telemetryLogs[len(m.telemetryLogs)-1000:]
			}
		}
		
		var formattedLogs []string
		for _, logLine := range m.telemetryLogs {
			formattedLogs = append(formattedLogs, formatLogLine(logLine))
		}
		m.telemetryViewport.SetContent(strings.Join(formattedLogs, "\n"))
		if wasAtBottom {
			m.telemetryViewport.GotoBottom()
		}
		return m, readClareLogsCmd(m.clareLogChan)

	case aniSkipCheckedMsg:
		m.aniSkipReady[msg.epNo] = msg.ready
		return m, nil

	case playbackFinishedMsg:
		debugLog("TUI playbackFinishedMsg: err=%v, tempLuaFile=%s, tempChaptersFile=%s", msg.err, msg.tempLuaFile, msg.tempChaptersFile)
		m.playbackActive = false
		if msg.tempLuaFile != "" {
			_ = os.Remove(msg.tempLuaFile)
			if m.tempLuaFile == msg.tempLuaFile {
				m.tempLuaFile = ""
			}
		}
		if msg.tempChaptersFile != "" {
			_ = os.Remove(msg.tempChaptersFile)
			if m.tempChaptersFile == msg.tempChaptersFile {
				m.tempChaptersFile = ""
			}
		}

		if msg.err == nil {
			_ = recordWatch(m.selectedShow.ID, m.selectedShow.Name, m.selectedEp)
			m.refreshHistory()
			if m.autoplay {
				m.triggerAutoplay = true
			}

			// Trigger background sync to AniList/MAL
			go func(malID string, epNo string) {
				time.Sleep(1 * time.Second)
				positions, err := loadPositions()
				if err == nil && malID != "" {
					if showState, ok := positions[malID]; ok {
						reqEp := parseEpisodeNumber(epNo)
						isCompleted := false
						for _, completedEp := range showState.CompletedEpisodes {
							if completedEp == reqEp {
								isCompleted = true
								break
							}
						}
						if isCompleted {
							SyncProgress(malID, epNo)
						}
					}
				}
			}(m.selectedShow.MALID, m.selectedEp)
		}

		m.state = stateSearchRunning
		m.loadingMsg = "Refreshing episode list..."
		return m, doFetchEpisodes(m.selectedShow.ID, "sub")

	case allStreamsResultMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.resolvedStreams = msg.streams
		var items []list.Item
		for _, s := range msg.streams {
			items = append(items, sourceItem{stream: s})
		}
		m.sourceList.SetItems(items)
		m.sourceList.Title = fmt.Sprintf("Episode %s Sources", msg.epNo)
		m.state = stateSourceSelect
		return m, nil

	case showDetailsResultMsg:
		if msg.err == nil {
			m.historyShowDetails[msg.showID] = msg.show
			if msg.show.Thumbnail != "" {
				if _, ok := m.coverArtCache[msg.showID]; !ok {
					m.coverArtCache[msg.showID] = "Loading..."
					return m, doFetchCoverArt(msg.showID, msg.show.Thumbnail, 16, 11)
				}
			}
		} else {
			debugLog("TUI showDetailsResultMsg error: %v", msg.err)
		}
		return m, nil

	case CoverArtLoadedMsg:
		if msg.Ansi != "" {
			m.coverArtCache[msg.ShowID] = msg.Ansi
		} else {
			m.coverArtCache[msg.ShowID] = ""
		}
		return m, nil

	case jikanMetadataMsg:
		if msg.err != nil {
			debugLog("TUI jikanMetadataMsg error: %v", msg.err)
			m.loadedJikanPages[msg.page] = false // Allow retry later
			return m, nil
		}
		// Save new page metadata to cache
		cacheData, _ := loadJikanCache(msg.malID)
		if cacheData == nil {
			cacheData = make(map[string]JikanEpInfo)
		}
		for k, v := range msg.metadata {
			m.episodeDetails[k] = v
			cacheData[k] = v
		}
		_ = saveJikanCache(msg.malID, cacheData)
		m.refreshEpisodeListItems()

		// Trigger lazy-loading of synopsis for the currently selected episode
		selectedItem := m.episodeList.SelectedItem()
		if selectedItem != nil && m.selectedShow.MALID != "" && m.selectedShow.MALID != "0" {
			if epItem, ok := selectedItem.(episodeItem); ok {
				if info, ok := m.episodeDetails[epItem.epNo]; !ok || info.Synopsis == "" {
					if !m.fetchingSynopsis[epItem.epNo] {
						m.fetchingSynopsis[epItem.epNo] = true
						return m, doFetchEpisodeSynopsis(m.selectedShow.MALID, epItem.epNo)
					}
				}
			}
		}
		return m, nil

	case episodeSynopsisMsg:
		if msg.err != nil {
			debugLog("TUI episodeSynopsisMsg error: %v", msg.err)
			return m, nil
		}
		if info, ok := m.episodeDetails[msg.epNo]; ok {
			info.Synopsis = msg.synopsis
			m.episodeDetails[msg.epNo] = info
		} else {
			m.episodeDetails[msg.epNo] = JikanEpInfo{
				Synopsis: msg.synopsis,
			}
		}
		cacheData, _ := loadJikanCache(m.selectedShow.MALID)
		if cacheData == nil {
			cacheData = make(map[string]JikanEpInfo)
		}
		info := cacheData[msg.epNo]
		info.Synopsis = msg.synopsis
		cacheData[msg.epNo] = info
		_ = saveJikanCache(m.selectedShow.MALID, cacheData)
		return m, nil

	case tea.MouseMsg:
		switch m.state {
		case stateSearchInput:
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		case stateShowSelect:
			var cmd tea.Cmd
			m.showList, cmd = m.showList.Update(msg)
			return m, cmd
		case stateEpisodeSelect:
			var cmd tea.Cmd
			m.episodeList, cmd = m.episodeList.Update(msg)
			return m, cmd
		case stateHistory:
			var cmd tea.Cmd
			m.historyList, cmd = m.historyList.Update(msg)
			return m, cmd
		case stateLogs:
			var cmd tea.Cmd
			m.telemetryViewport, cmd = m.telemetryViewport.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		debugLog("TUI KeyMsg: key=%s, state=%d", msg.String(), m.state)

		// Intercept global media controls if playback is active and user is not typing/filtering
		isFiltering := (m.state == stateEpisodeSelect && m.episodeList.FilterState() == list.Filtering) ||
			(m.state == stateShowSelect && m.showList.FilterState() == list.Filtering) ||
			(m.state == stateHistory && m.historyList.FilterState() == list.Filtering)

		if m.playbackActive && m.state != stateSearchInput && !isFiltering {
			switch msg.String() {
			case "a":
				m.autoplay = !m.autoplay
				_ = saveConfig(m.getConfig())
				return m, nil
			case "s":
				m.autoskip = !m.autoskip
				_ = saveConfig(m.getConfig())
				return m, nil
			case "f":
				m.skipFillers = !m.skipFillers
				_ = saveConfig(m.getConfig())
				return m, nil
			case "p", " ":
				_ = executeMpvAction([]interface{}{"cycle", "pause"})
				return m, nil
			case "[":
				_ = executeMpvAction([]interface{}{"seek", -10})
				return m, nil
			case "]":
				_ = executeMpvAction([]interface{}{"seek", 10})
				return m, nil
			case "-":
				_ = executeMpvAction([]interface{}{"add", "volume", -5})
				return m, nil
			case "+", "=":
				_ = executeMpvAction([]interface{}{"add", "volume", 5})
				return m, nil
			case "x":
				if m.activeCmd != nil && m.activeCmd.Process != nil {
					_ = m.activeCmd.Process.Kill()
				}
				return m, nil
			}
		}

		// Let active filtering lists handle all keystrokes directly
		if m.state == stateEpisodeSelect && m.episodeList.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.episodeList, cmd = m.episodeList.Update(msg)
			return m, cmd
		}
		if m.state == stateShowSelect && m.showList.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.showList, cmd = m.showList.Update(msg)
			return m, cmd
		}
		if m.state == stateHistory && m.historyList.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.historyList, cmd = m.historyList.Update(msg)
			return m, cmd
		}
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			// Don't quit with 'q' if we are typing in text input
			if m.state != stateSearchInput {
				return m, tea.Quit
			}
		case "1":
			if m.state != stateSearchInput {
				m.state = stateHistory
				m.recalculateSizes()
				return m, nil
			}
		case "2":
			if m.state != stateSearchInput {
				m.enterSearchState()
				return m, nil
			}
		case "3":
			if m.state != stateSearchInput {
				m.state = stateLogs
				m.recalculateSizes()
				return m, nil
			}
		case "4":
			if m.state != stateSearchInput {
				m.state = stateConfig
				m.recalculateSizes()
				return m, nil
			}
		case "tab":
			if m.state != stateSearchInput {
				if m.state == stateHistory {
					m.enterSearchState()
				} else if m.state == stateSearchInput || m.state == stateSearchRunning || m.state == stateShowSelect {
					m.state = stateLogs
					m.recalculateSizes()
				} else if m.state == stateLogs {
					m.state = stateConfig
					m.recalculateSizes()
				} else {
					m.state = stateHistory
					m.recalculateSizes()
				}
				return m, nil
			}
		}

		switch m.state {

		case stateHistory:
			switch msg.String() {
			case "enter":
				selected, ok := m.historyList.SelectedItem().(historyItem)
				if ok {
					m.selectedShow = AnimeShow{
						ID:   selected.showID,
						Name: selected.showName,
					}
					m.state = stateSearchRunning
					m.loadingMsg = fmt.Sprintf("Fetching episodes for %s...", m.selectedShow.Name)
					return m, doFetchEpisodes(selected.showID, "sub")
				}
			case "c", "C":
				m.showCompleted = !m.showCompleted
				m.applyHistoryFilter()
				return m, nil
			case "w", "W":
				if selected, ok := m.historyList.SelectedItem().(historyItem); ok {
					m.toggleShowCompleted(selected.showID)
				}
				return m, nil
			case "d", "D":
				if selected, ok := m.historyList.SelectedItem().(historyItem); ok {
					if rawHist, err := loadHistory(); err == nil {
						var filtered []HistoryEntry
						for _, h := range rawHist {
							if h.ShowID != selected.showID {
								filtered = append(filtered, h)
							}
						}
						_ = saveHistory(filtered)
					}
					m.refreshHistory()
				}
				return m, nil
			case "s", "/":
				m.enterSearchState()
				return m, nil
			case "left", "h":
				m.detailsScrollOffset--
				return m, nil
			case "right", "l":
				m.detailsScrollOffset++
				return m, nil
			}
			var cmd tea.Cmd
			m.historyList, cmd = m.historyList.Update(msg)
			
			// Trigger details fetch for currently highlighted history item if not loaded
			if selected, ok := m.historyList.SelectedItem().(historyItem); ok {
				if selected.showID != m.lastSelectedShowID {
					m.detailsScrollOffset = 0
					m.lastSelectedShowID = selected.showID
				}
				var cmds []tea.Cmd
				if _, loaded := m.historyShowDetails[selected.showID]; !loaded {
					if show, _, found := loadShowCache(selected.showID); found {
						m.historyShowDetails[selected.showID] = show
						if show.Thumbnail != "" {
							if _, ok := m.coverArtCache[selected.showID]; !ok {
								m.coverArtCache[selected.showID] = "Loading..."
								cmds = append(cmds, doFetchCoverArt(selected.showID, show.Thumbnail, 16, 11))
							}
						}
					} else {
						m.historyShowDetails[selected.showID] = AnimeShow{
							ID:          selected.showID,
							Name:        selected.showName,
							Description: "Loading details...",
						}
						cmds = append(cmds, doFetchShowDetails(selected.showID))
					}
				} else {
					show := m.historyShowDetails[selected.showID]
					if show.Thumbnail != "" {
						if _, ok := m.coverArtCache[selected.showID]; !ok {
							m.coverArtCache[selected.showID] = "Loading..."
							cmds = append(cmds, doFetchCoverArt(selected.showID, show.Thumbnail, 16, 11))
						}
					}
				}
				if len(cmds) > 0 {
					return m, tea.Batch(append(cmds, cmd)...)
				}
			}
			return m, cmd

		case stateSearchInput:
			switch msg.String() {
			case "up":
				if len(m.searchHistory) > 0 && m.searchHistoryIndex < len(m.searchHistory)-1 {
					m.searchHistoryIndex++
					m.searchInput.SetValue(m.searchHistory[m.searchHistoryIndex])
					m.searchInput.SetCursor(len(m.searchHistory[m.searchHistoryIndex]))
				}
				return m, nil
			case "down":
				if m.searchHistoryIndex > 0 {
					m.searchHistoryIndex--
					m.searchInput.SetValue(m.searchHistory[m.searchHistoryIndex])
					m.searchInput.SetCursor(len(m.searchHistory[m.searchHistoryIndex]))
				} else if m.searchHistoryIndex == 0 {
					m.searchHistoryIndex = -1
					m.searchInput.SetValue("")
				}
				return m, nil
			case "enter":
				query := strings.TrimSpace(m.searchInput.Value())
				if query != "" {
					_ = recordSearch(query)
					m.state = stateSearchRunning
					m.loadingMsg = fmt.Sprintf("Searching for %q...", query)
					return m, doSearch(query, "sub")
				}
			case "esc":
				if len(m.historyItems) > 0 {
					m.state = stateHistory
				} else {
					return m, tea.Quit
				}
			default:
				m.searchHistoryIndex = -1
			}
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd

		case stateShowSelect:
			switch msg.String() {
			case "enter":
				selected, ok := m.showList.SelectedItem().(showItem)
				if ok {
					m.selectedShow = selected.show
					m.state = stateSearchRunning
					m.loadingMsg = fmt.Sprintf("Fetching episodes for %s...", m.selectedShow.Name)
					return m, doFetchEpisodes(selected.show.ID, "sub")
				}
			case "esc":
				m.state = stateSearchInput
				m.searchInput.Focus()
				return m, nil
			case "left", "h":
				m.detailsScrollOffset--
				return m, nil
			case "right", "l":
				m.detailsScrollOffset++
				return m, nil
			}
			var cmd tea.Cmd
			m.showList, cmd = m.showList.Update(msg)
			if selected, ok := m.showList.SelectedItem().(showItem); ok {
				if selected.show.ID != m.lastSelectedShowID {
					m.detailsScrollOffset = 0
					m.lastSelectedShowID = selected.show.ID
				}
				if selected.show.Thumbnail != "" {
					if _, ok := m.coverArtCache[selected.show.ID]; !ok {
						m.coverArtCache[selected.show.ID] = "Loading..."
						return m, tea.Batch(cmd, doFetchCoverArt(selected.show.ID, selected.show.Thumbnail, 16, 11))
					}
				}
			}
			return m, cmd

		case stateEpisodeSelect:
			switch msg.String() {
			case "tab":
				m.showTelemetry = !m.showTelemetry
				return m, nil
			case "left", "h":
				m.detailsScrollOffset--
				return m, nil
			case "right", "l":
				m.detailsScrollOffset++
				return m, nil
			case "a":
				m.autoplay = !m.autoplay
				return m, nil
			case "m":
				m.mode = nextMode(m.mode)
				return m, nil
			case "enter":
				selected, ok := m.episodeList.SelectedItem().(episodeItem)
				if ok {
					m.selectedEp = selected.epNo
					m.state = stateSearchRunning
					m.loadingMsg = fmt.Sprintf("Resolving stream sources for Episode %s...", selected.epNo)
					return m, doFetchAllStreams(m.selectedShow.ID, m.mode, selected.epNo)
				}
			case "esc":
				// If we came from history, go back to history. Else, show selection.
				if len(m.showItems) > 1 {
					m.state = stateShowSelect
					if selected, ok := m.showList.SelectedItem().(showItem); ok {
						if selected.show.Thumbnail != "" {
							if _, ok := m.coverArtCache[selected.show.ID]; !ok {
								m.coverArtCache[selected.show.ID] = "Loading..."
								return m, tea.Batch(doFetchCoverArt(selected.show.ID, selected.show.Thumbnail, 16, 11))
							}
						}
					}
				} else if len(m.historyItems) > 0 {
					m.state = stateHistory
					if selected, ok := m.historyList.SelectedItem().(historyItem); ok {
						var cmds []tea.Cmd
						if _, loaded := m.historyShowDetails[selected.showID]; !loaded {
							if show, _, found := loadShowCache(selected.showID); found {
								m.historyShowDetails[selected.showID] = show
								if show.Thumbnail != "" {
									if _, ok := m.coverArtCache[selected.showID]; !ok {
										m.coverArtCache[selected.showID] = "Loading..."
										cmds = append(cmds, doFetchCoverArt(selected.showID, show.Thumbnail, 16, 11))
									}
								}
							} else {
								m.historyShowDetails[selected.showID] = AnimeShow{
									ID:          selected.showID,
									Name:        selected.showName,
									Description: "Loading details...",
								}
								cmds = append(cmds, doFetchShowDetails(selected.showID))
							}
						} else {
							show := m.historyShowDetails[selected.showID]
							if show.Thumbnail != "" {
								if _, ok := m.coverArtCache[selected.showID]; !ok {
									m.coverArtCache[selected.showID] = "Loading..."
									cmds = append(cmds, doFetchCoverArt(selected.showID, show.Thumbnail, 16, 11))
								}
							}
						}
						if len(cmds) > 0 {
							return m, tea.Batch(cmds...)
						}
					}
				} else {
					m.state = stateSearchInput
					m.searchInput.Focus()
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.episodeList, cmd = m.episodeList.Update(msg)
			
			var cmds []tea.Cmd
			cmds = append(cmds, cmd)

			// Trigger cover art fetch if not cached
			if m.selectedShow.Thumbnail != "" {
				if _, ok := m.coverArtCache[m.selectedShow.ID]; !ok {
					m.coverArtCache[m.selectedShow.ID] = "Loading..."
					cmds = append(cmds, doFetchCoverArt(m.selectedShow.ID, m.selectedShow.Thumbnail, 16, 11))
				}
			}

			// Trigger lazy-loading of synopsis for selected episode
			selectedItem := m.episodeList.SelectedItem()
			if selectedItem != nil && m.selectedShow.MALID != "" && m.selectedShow.MALID != "0" {
				if epItem, ok := selectedItem.(episodeItem); ok {
					if info, ok := m.episodeDetails[epItem.epNo]; !ok || info.Synopsis == "" {
						if !m.fetchingSynopsis[epItem.epNo] {
							m.fetchingSynopsis[epItem.epNo] = true
							cmds = append(cmds, doFetchEpisodeSynopsis(m.selectedShow.MALID, epItem.epNo))
						}
					}
				}
			}

			// Lazy load metadata page for currently selected episode if needed
			if item := m.episodeList.SelectedItem(); item != nil {
				if epItem, ok := item.(episodeItem); ok {
					if epItem.epNo != m.selectedEp {
						m.detailsScrollOffset = 0
						m.selectedEp = epItem.epNo
					}
					var val int
					fmt.Sscanf(epItem.epNo, "%d", &val)
					if val > 0 {
						page := (val - 1) / 100 + 1
						if !m.loadedJikanPages[page] {
							m.loadedJikanPages[page] = true
							cmds = append(cmds, doFetchJikanMetadata(m.selectedShow.MALID, page))
						}
					}
					if _, checked := m.aniSkipReady[epItem.epNo]; !checked && m.selectedShow.MALID != "" {
						m.aniSkipReady[epItem.epNo] = false
						cmds = append(cmds, doCheckAniSkip(m.selectedShow.MALID, epItem.epNo))
					}
					prefetchEpisodeStream(m.selectedShow.ID, m.mode, epItem.epNo, m.quality)
					idx := m.episodeList.Index()
					if idx+1 < len(m.episodeItems) {
						if nextEpItem, ok := m.episodeItems[idx+1].(episodeItem); ok {
							prefetchEpisodeStream(m.selectedShow.ID, m.mode, nextEpItem.epNo, m.quality)
						}
					}
				}
			}
			return m, tea.Batch(cmds...)

		case stateSourceSelect:
			switch msg.String() {
			case "enter":
				selected, ok := m.sourceList.SelectedItem().(sourceItem)
				if ok {
					m.state = statePlaybackPreparing
					m.loadingMsg = fmt.Sprintf("Preparing playback from %s (%s)...", selected.stream.SourceName, selected.stream.Quality)

					// Seed cache so resolveStreamURL uses this exact URL
					cacheKey := fmt.Sprintf("%s-%s-%s-%s", m.selectedShow.ID, m.mode, m.selectedEp, m.quality)
					streamCacheMu.Lock()
					streamCache[cacheKey] = selected.stream.URL
					streamCacheMu.Unlock()

					return m, doPreparePlayback(m.selectedShow, m.selectedEp, m.mode, m.quality, m.download)
				}
			case "esc":
				m.state = stateEpisodeSelect
				return m, nil
			}
			var cmd tea.Cmd
			m.sourceList, cmd = m.sourceList.Update(msg)
			return m, cmd


		case stateLogs:
			switch msg.String() {
			case "esc":
				m.state = stateHistory
				m.recalculateSizes()
				return m, nil
			}
			var cmd tea.Cmd
			m.telemetryViewport, cmd = m.telemetryViewport.Update(msg)
			return m, cmd

		case stateConfig:
			switch msg.String() {
			case "esc":
				m.state = stateHistory
				m.recalculateSizes()
				return m, nil
			case "up", "k":
				if m.configCursor > 0 {
					m.configCursor--
				}
				return m, nil
			case "down", "j":
				if m.configCursor < 4 {
					m.configCursor++
				}
				return m, nil
			case "enter", " ", "right", "l", "left", "h":
				cfg := m.getConfig()
				switch m.configCursor {
				case 0:
					cfg.Autoplay = !cfg.Autoplay
				case 1:
					cfg.Autoskip = !cfg.Autoskip
				case 2:
					cfg.SkipFillers = !cfg.SkipFillers
				case 3:
					if cfg.PreferredMode == "sub" {
						cfg.PreferredMode = "dub"
					} else if cfg.PreferredMode == "dub" {
						cfg.PreferredMode = "dual"
					} else {
						cfg.PreferredMode = "sub"
					}
				case 4:
					qualities := []string{"best", "1080p", "720p", "480p", "360p"}
					idx := -1
					for i, q := range qualities {
						if q == cfg.PreferredQuality {
							idx = i
							break
						}
					}
					nextIdx := (idx + 1) % len(qualities)
					cfg.PreferredQuality = qualities[nextIdx]
				}
				_ = saveConfig(cfg)
				m.autoplay = cfg.Autoplay
				m.autoskip = cfg.Autoskip
				m.skipFillers = cfg.SkipFillers
				m.mode = cfg.PreferredMode
				m.quality = cfg.PreferredQuality
				return m, nil
			}
			return m, nil

		case stateError:
			switch msg.String() {
			case "enter", "esc":
				m.state = stateSearchInput
				m.searchInput.Focus()
				m.err = nil
				return m, nil
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	var s strings.Builder

	// Top Banner
	s.WriteString(titleStyle.Render(" クレア "))
	s.WriteString("\n\n")

	// Navigation Tabs
	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#1f2335")).
		Background(lipgloss.Color("#7aa2f7")).
		Padding(0, 2).
		MarginRight(1)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a9b1d6")).
		Background(lipgloss.Color("#24283b")).
		Padding(0, 2).
		MarginRight(1)

	var tabs []string
	showTabs := false
	if m.state == stateHistory {
		tabs = append(tabs, activeTabStyle.Render("Continue Watching [1]"))
		tabs = append(tabs, inactiveTabStyle.Render("Search [2]"))
		tabs = append(tabs, inactiveTabStyle.Render("Logs [3]"))
		tabs = append(tabs, inactiveTabStyle.Render("Config [4]"))
		showTabs = true
	} else if m.state == stateSearchInput || m.state == stateSearchRunning || m.state == stateShowSelect {
		tabs = append(tabs, inactiveTabStyle.Render("Continue Watching [1]"))
		tabs = append(tabs, activeTabStyle.Render("Search [2]"))
		tabs = append(tabs, inactiveTabStyle.Render("Logs [3]"))
		tabs = append(tabs, inactiveTabStyle.Render("Config [4]"))
		showTabs = true
	} else if m.state == stateLogs {
		tabs = append(tabs, inactiveTabStyle.Render("Continue Watching [1]"))
		tabs = append(tabs, inactiveTabStyle.Render("Search [2]"))
		tabs = append(tabs, activeTabStyle.Render("Logs [3]"))
		tabs = append(tabs, inactiveTabStyle.Render("Config [4]"))
		showTabs = true
	} else if m.state == stateConfig {
		tabs = append(tabs, inactiveTabStyle.Render("Continue Watching [1]"))
		tabs = append(tabs, inactiveTabStyle.Render("Search [2]"))
		tabs = append(tabs, inactiveTabStyle.Render("Logs [3]"))
		tabs = append(tabs, activeTabStyle.Render("Config [4]"))
		showTabs = true
	}

	listHeight := m.dynamicListHeight()

	if showTabs {
		s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...) + "\n\n")
	}

	bodyStyle := lipgloss.NewStyle().Height(listHeight)

	switch m.state {
	case stateHistory:
		if m.width >= 80 {
			leftWidth := m.width / 2
			if leftWidth < 35 {
				leftWidth = 35
			}
			rightWidth := m.width - leftWidth - 4
			if rightWidth < 10 {
				rightWidth = 10
			}

			leftView := m.historyList.View()
			var rightView string
			if selected, ok := m.historyList.SelectedItem().(historyItem); ok {
				art := m.coverArtCache[selected.showID]
				if show, loaded := m.historyShowDetails[selected.showID]; loaded {
					rightView = m.renderShowDetailsPanel(show, art, rightWidth, listHeight)
				} else {
					tempShow := AnimeShow{ID: selected.showID, Name: selected.showName, Description: "Loading details..."}
					rightView = m.renderShowDetailsPanel(tempShow, art, rightWidth, listHeight)
				}
			} else {
				rightView = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("#7aa2f7")).
					Padding(1, 2).
					Width(rightWidth).
					Height(listHeight).
					Render("No show selected.")
			}

			s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView))
		} else {
			s.WriteString(m.historyList.View())
		}
		s.WriteString("\n\n" + helpStyle("s: search  enter: resume  c: toggle completed  d: remove  q: quit"))

	case stateSearchInput:
		var bodyBuf strings.Builder
		bodyBuf.WriteString(accentColorStyle.Render("Search Anime:") + "\n\n")
		bodyBuf.WriteString(m.searchInput.View() + "\n\n")

		if len(m.searchHistory) > 0 {
			bodyBuf.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Bold(true).Render("Recent Searches:") + "\n")
			for i, q := range m.searchHistory {
				if i == m.searchHistoryIndex {
					bodyBuf.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7")).Render(fmt.Sprintf("  ❯ %s", q)) + "\n")
				} else {
					bodyBuf.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")).Render(fmt.Sprintf("    %s", q)) + "\n")
				}
			}
			bodyBuf.WriteString("\n")
		}
		s.WriteString(bodyStyle.Render(bodyBuf.String()))
		s.WriteString("\n\n" + helpStyle("enter: search | up/down: browse history | esc: cancel"))

	case stateSearchRunning:
		bodyContent := fmt.Sprintf("%s %s\n", m.spinner.View(), m.loadingMsg)
		s.WriteString(bodyStyle.Render(bodyContent))
		s.WriteString("\n\n" + helpStyle("Please wait..."))

	case stateShowSelect:
		if m.width >= 80 {
			leftWidth := m.width / 2
			if leftWidth < 35 {
				leftWidth = 35
			}
			rightWidth := m.width - leftWidth - 4
			if rightWidth < 10 {
				rightWidth = 10
			}

			leftView := m.showList.View()
			var rightView string
			if selected, ok := m.showList.SelectedItem().(showItem); ok {
				art := m.coverArtCache[selected.show.ID]
				rightView = m.renderShowDetailsPanel(selected.show, art, rightWidth, listHeight)
			} else {
				rightView = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("#7aa2f7")).
					Padding(1, 2).
					Width(rightWidth).
					Height(listHeight).
					Render("No show selected.")
			}

			s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView))
		} else {
			s.WriteString(m.showList.View())
		}
		s.WriteString("\n\n" + helpStyle("enter: select show | esc: back | q: quit"))

	case stateEpisodeSelect:
		if m.width >= 80 {
			leftWidth := m.width / 2
			if leftWidth < 35 {
				leftWidth = 35
			}
			rightWidth := m.width - leftWidth - 4
			if rightWidth < 10 {
				rightWidth = 10
			}

			leftView := m.episodeList.View()
			rightView := m.renderEpisodeDetailsPanel(rightWidth, listHeight)

			s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView))
		} else {
			s.WriteString(m.episodeList.View())
		}
		autoplayStr := "autoplay: OFF"
		if m.autoplay {
			autoplayStr = "autoplay: ON"
		}
		modeStr := strings.ToUpper(m.mode)
		s.WriteString("\n\n" + helpStyle(fmt.Sprintf("enter: play | a: toggle autoplay (%s) | m: toggle mode (mode: %s) | esc: back | q: quit", autoplayStr, modeStr)))

	case stateSourceSelect:
		s.WriteString(m.sourceList.View())
		s.WriteString("\n\n" + helpStyle("enter: play with selected source | esc: back | q: quit"))

	case statePlaybackPreparing:
		bodyContent := fmt.Sprintf("%s %s\n", m.spinner.View(), m.loadingMsg)
		s.WriteString(bodyStyle.Render(bodyContent))
		s.WriteString("\n\n" + helpStyle("Preparing playback..."))

	case statePlaybackActive:
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#bb9af7"))
		statusStr := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9ece6a")).Render("▶ PLAYING")
		if m.mpvStatus.Paused {
			statusStr = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e0af68")).Render("⏸ PAUSED")
		}

		var activeBody strings.Builder
		activeBody.WriteString(titleStyle.Render(fmt.Sprintf("Playing: %s — Episode %s", m.selectedShow.Name, m.selectedEp)) + "\n\n")
		activeBody.WriteString(fmt.Sprintf("Status: %s\n\n", statusStr))

		if m.mpvStatus.Duration > 0 {
			pct := m.mpvStatus.PlaybackTime / m.mpvStatus.Duration
			bar := renderSmoothProgressBar(pct, 30)
			timeStr := fmt.Sprintf("%s / %s", formatTime(m.mpvStatus.PlaybackTime), formatTime(m.mpvStatus.Duration))
			activeBody.WriteString(fmt.Sprintf("[%s]  %s\n\n", bar, timeStr))
		} else {
			activeBody.WriteString("Loading playback time...\n\n")
		}

		activeBody.WriteString(fmt.Sprintf("Volume: %d%%\n\n", int(m.mpvStatus.Volume)))
		s.WriteString(bodyStyle.Render(activeBody.String()))
		s.WriteString("\n\n" + helpStyle("space: pause/resume  left/right: seek 10s  up/down: volume  esc/q: stop playback"))

	case stateLogs:
		s.WriteString(m.telemetryViewport.View())
		s.WriteString("\n\n" + helpStyle("1: history | 2: search | q: quit"))

	case stateConfig:
		var cfgStrings []string
		cfg := m.getConfig()
		
		options := []struct {
			name  string
			value string
		}{
			{"Autoplay Next Episode", formatBool(cfg.Autoplay)},
			{"Auto-skip Openings/Endings", formatBool(cfg.Autoskip)},
			{"Automatically Skip Fillers", formatBool(cfg.SkipFillers)},
			{"Preferred Translation Mode", strings.ToUpper(cfg.PreferredMode)},
			{"Preferred Stream Quality", strings.ToUpper(cfg.PreferredQuality)},
		}

		for i, opt := range options {
			cursor := "  "
			itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#c0caf5"))
			if i == m.configCursor {
				cursor = "❯ "
				itemStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7aa2f7"))
			}
			cfgStrings = append(cfgStrings, fmt.Sprintf("%s%-30s : %s", cursor, itemStyle.Render(opt.name), opt.value))
		}

		infoCardStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#565f89")).
			Padding(1, 2).
			Width(m.width - 6)

		var infoCard strings.Builder
		infoCard.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#bb9af7")).Render("◆ AniList / MAL Sync Configuration ◆") + "\n\n")
		infoCard.WriteString("To enable automatic progress synchronization to AniList or MyAnimeList,\n")
		infoCard.WriteString("set your tokens inside the config file at:\n")
		infoCard.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff")).Render("~/.config/clare/config.json") + "\n\n")
		infoCard.WriteString("Example configuration:\n")
		infoCard.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")).Render(`{
  "autoplay": true,
  "autoskip": true,
  "skip_fillers": false,
  "anilist_token": "YOUR_ANILIST_ACCESS_TOKEN",
  "mal_token": "YOUR_MAL_ACCESS_TOKEN"
}`) + "\n\n")
		infoCard.WriteString("Note: Run clare's log tab [3] during sync to check status/verify successful API requests.")

		bodyContent := fmt.Sprintf(
			"◆ TUI USER CONFIGURATION ◆\n\n%s\n\n%s",
			strings.Join(cfgStrings, "\n"),
			infoCardStyle.Render(infoCard.String()),
		)

		s.WriteString(bodyStyle.Render(bodyContent))
		s.WriteString("\n\n" + helpStyle("enter/space: toggle/cycle | up/down: navigate | esc: back"))

	case stateError:
		s.WriteString(errorStyle.Render("Error encountered:") + "\n\n")
		s.WriteString(fmt.Sprintf("  %v\n\n", m.err))
		s.WriteString(helpStyle("press enter or esc to return to search"))
	}

	bodyStr := s.String()
	if m.playbackActive {
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#bb9af7"))
		statusStr := "▶ PLAYING"
		if m.mpvStatus.Paused {
			statusStr = "⏸ PAUSED"
		}
		
		var pb string
		if m.mpvStatus.Duration > 0 {
			pct := m.mpvStatus.PlaybackTime / m.mpvStatus.Duration
			bar := renderSmoothProgressBar(pct, 20)
			timeStr := fmt.Sprintf("%s/%s", formatTime(m.mpvStatus.PlaybackTime), formatTime(m.mpvStatus.Duration))
			pb = fmt.Sprintf("[%s] %s", bar, timeStr)
		} else {
			pb = "Loading..."
		}

		playerBorder := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#bb9af7")).
			Padding(0, 1).
			Width(m.width - 2)

		formatCheckbox := func(name string, val bool) string {
			box := "[ ]"
			if val {
				box = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")).Render("[✔]")
			}
			return fmt.Sprintf("%s %s", box, name)
		}

		autoplayToggle := formatCheckbox("Autoplay (a)", m.autoplay)
		autoskipToggle := formatCheckbox("Auto-Skip (s)", m.autoskip)
		skipFillersToggle := formatCheckbox("Skip Fillers (f)", m.skipFillers)

		playerContent := fmt.Sprintf("%s %s  %s  Vol: %d%%\n%s  •  %s  •  %s", 
			statusStr, 
			titleStyle.Render(fmt.Sprintf("%s - Ep %s", m.playingShow.Name, m.playingEp)),
			pb,
			int(m.mpvStatus.Volume),
			autoplayToggle,
			autoskipToggle,
			skipFillersToggle,
		)

		playerView := playerBorder.Render(playerContent)
		playerHeight := lipgloss.Height(playerView)
		bodyHeight := lipgloss.Height(bodyStr)
		
		padHeight := m.height - bodyHeight - playerHeight
		if padHeight > 0 {
			bodyStr += strings.Repeat("\n", padHeight)
		}
		bodyStr += playerView
	}

	return bodyStr
}

func formatTime(seconds float64) string {
	if seconds <= 0 {
		return "00:00"
	}
	h := int(seconds) / 3600
	m := (int(seconds) % 3600) / 60
	s := int(seconds) % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func helpStyle(val string) string {
	return grayColorStyle.Render(val)
}

// Async search command
func doSearch(query, mode string) tea.Cmd {
	return func() tea.Msg {
		shows, err := searchAnime(query, mode)
		return searchResultMsg{shows: shows, err: err}
	}
}

// Async episodes fetch command
func doFetchEpisodes(showID, mode string) tea.Cmd {
	return func() tea.Msg {
		show, eps, err := fetchEpisodeList(showID, mode)
		return episodesResultMsg{show: show, episodes: eps, err: err}
	}
}

// Async resolve streams and player build command
func doPreparePlayback(selectedShow AnimeShow, epNo, mode, quality string, download bool) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		var tempLua string
		var tempChapters string
		var err error

		if download {
			// Resolve URL and build download command
			var stream string
			if mode == "dual" || mode == "sub" {
				stream, err = resolveStreamURL(selectedShow.ID, "sub", epNo, quality)
			} else {
				stream, err = resolveStreamURL(selectedShow.ID, "dub", epNo, quality)
			}
			if err != nil {
				return resolvedPlaybackMsg{err: err}
			}
			cmd = downloadCmd(stream, selectedShow.Name, epNo)
			return resolvedPlaybackMsg{cmd: cmd, err: nil}
		}

		if mode == "dual" {
			subStream, errSub := resolveStreamURL(selectedShow.ID, "sub", epNo, quality)
			dubStream, errDub := resolveStreamURL(selectedShow.ID, "dub", epNo, quality)

			if errSub != nil {
				if errDub != nil {
					return resolvedPlaybackMsg{err: fmt.Errorf("failed to resolve dual streams: sub (%v), dub (%v)", errSub, errDub)}
				}
				debugLog("doPreparePlayback: sub failed (%v), falling back to dub-only", errSub)
				cmd, tempLua, tempChapters, err = playSingleCmd(dubStream, selectedShow.Name, epNo, selectedShow.MALID, selectedShow.Duration)
			} else if errDub != nil {
				debugLog("doPreparePlayback: dub failed (%v), falling back to sub-only", errDub)
				cmd, tempLua, tempChapters, err = playSingleCmd(subStream, selectedShow.Name, epNo, selectedShow.MALID, selectedShow.Duration)
				if err == nil {
					return resolvedPlaybackMsg{cmd: cmd, tempLuaFile: tempLua, tempChaptersFile: tempChapters, warning: fmt.Sprintf("⚠ Dub unavailable (%v) — playing sub only", errDub)}
				}
			} else {
				debugLog("doPreparePlayback: both streams resolved, launching dual-audio")
				cmd, tempLua, tempChapters, err = playDualCmd(subStream, dubStream, selectedShow.Name, epNo, selectedShow.MALID, selectedShow.Duration)
			}
		} else if mode == "dub" {
			dubStream, errDub := resolveStreamURL(selectedShow.ID, "dub", epNo, quality)
			if errDub != nil {
				return resolvedPlaybackMsg{err: errDub}
			}
			cmd, tempLua, tempChapters, err = playSingleCmd(dubStream, selectedShow.Name, epNo, selectedShow.MALID, selectedShow.Duration)
		} else {
			subStream, errSub := resolveStreamURL(selectedShow.ID, "sub", epNo, quality)
			if errSub != nil {
				return resolvedPlaybackMsg{err: errSub}
			}
			cmd, tempLua, tempChapters, err = playSingleCmd(subStream, selectedShow.Name, epNo, selectedShow.MALID, selectedShow.Duration)
		}

		return resolvedPlaybackMsg{cmd: cmd, tempLuaFile: tempLua, tempChaptersFile: tempChapters, err: err}
	}
}

func waitForExitCmd(cmd *exec.Cmd, tempLuaFile, tempChaptersFile string) tea.Cmd {
	return func() tea.Msg {
		err := cmd.Wait()
		return playbackFinishedMsg{
			err:              err,
			tempLuaFile:      tempLuaFile,
			tempChaptersFile: tempChaptersFile,
		}
	}
}

func readClareLogsCmd(logChan chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-logChan
		if !ok {
			return nil
		}
		return clareLogMsg(line)
	}
}

func tailLogFile(logChan chan string) {
	time.Sleep(500 * time.Millisecond)
	dir := os.Getenv("CLARE_STATE_DIR")
	if dir == "" {
		stateHome := os.Getenv("XDG_STATE_HOME")
		if stateHome == "" {
			home, _ := os.UserHomeDir()
			stateHome = filepath.Join(home, ".local", "state")
		}
		dir = filepath.Join(stateHome, "clare")
	}
	logFile := filepath.Join(dir, "debug.log")

	// Ensure the dir exists and we can touch the file
	_ = os.MkdirAll(dir, 0755)
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	_, _ = f.Seek(0, io.SeekEnd)
	reader := bufio.NewReader(f)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			break
		}
		logChan <- strings.TrimSuffix(line, "\n")
	}
}

func doCheckAniSkip(malID, epNo string) tea.Cmd {
	return func() tea.Msg {
		times := fetchAniSkipTimes(malID, epNo, 1440.0)
		return aniSkipCheckedMsg{epNo: epNo, ready: len(times) > 0}
	}
}

func doFetchJikanMetadata(malID string, page int) tea.Cmd {
	return func() tea.Msg {
		if malID == "" || malID == "0" {
			return jikanMetadataMsg{malID: malID, page: page, err: fmt.Errorf("no MAL ID")}
		}
		client := newLoggingHttpClient(8 * time.Second)
		url := fmt.Sprintf("https://api.jikan.moe/v4/anime/%s/episodes?page=%d", malID, page)
		resp, err := client.Get(url)
		if err != nil {
			return jikanMetadataMsg{malID: malID, page: page, err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return jikanMetadataMsg{malID: malID, page: page, err: fmt.Errorf("status %d", resp.StatusCode)}
		}
		var res struct {
			Data []struct {
				MalID    int    `json:"mal_id"`
				Title    string `json:"title"`
				Aired    string `json:"aired"`
				Synopsis string `json:"synopsis"`
				Filler   bool   `json:"filler"`
				Recap    bool   `json:"recap"`
			} `json:"data"`
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return jikanMetadataMsg{malID: malID, page: page, err: err}
		}
		if err := json.Unmarshal(body, &res); err != nil {
			return jikanMetadataMsg{malID: malID, page: page, err: err}
		}
		metadata := make(map[string]JikanEpInfo)
		for _, d := range res.Data {
			epNum := fmt.Sprintf("%d", d.MalID)
			airedStr := d.Aired
			if len(airedStr) >= 10 {
				airedStr = airedStr[:10]
			}
			metadata[epNum] = JikanEpInfo{
				Title:    d.Title,
				Aired:    airedStr,
				Synopsis: d.Synopsis,
				Filler:   d.Filler,
				Recap:    d.Recap,
			}
		}
		return jikanMetadataMsg{malID: malID, page: page, metadata: metadata}
	}
}

type episodeSynopsisMsg struct {
	epNo     string
	synopsis string
	err      error
}

func doFetchEpisodeSynopsis(malID, epNo string) tea.Cmd {
	return func() tea.Msg {
		if malID == "" || malID == "0" || epNo == "" {
			return episodeSynopsisMsg{epNo: epNo, err: fmt.Errorf("invalid arguments")}
		}
		epID, err := strconv.Atoi(epNo)
		if err != nil || epID <= 0 {
			return episodeSynopsisMsg{epNo: epNo, err: fmt.Errorf("invalid episode ID")}
		}

		client := newLoggingHttpClient(8 * time.Second)
		url := fmt.Sprintf("https://api.jikan.moe/v4/anime/%s/episodes/%d", malID, epID)
		resp, err := client.Get(url)
		if err != nil {
			return episodeSynopsisMsg{epNo: epNo, err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return episodeSynopsisMsg{epNo: epNo, err: fmt.Errorf("status %d", resp.StatusCode)}
		}

		var res struct {
			Data struct {
				Synopsis string `json:"synopsis"`
			} `json:"data"`
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return episodeSynopsisMsg{epNo: epNo, err: err}
		}
		if err := json.Unmarshal(body, &res); err != nil {
			return episodeSynopsisMsg{epNo: epNo, err: err}
		}

		return episodeSynopsisMsg{epNo: epNo, synopsis: res.Data.Synopsis}
	}
}

func (m *model) refreshEpisodeListItems() {
	lastEp := ""
	rawHist, _ := loadHistory()
	for _, h := range rawHist {
		if h.ShowID == m.selectedShow.ID {
			lastEp = h.Episode
			break
		}
	}

	nextEp := ""
	var showState *ShowState
	positions, _ := loadPositions()
	if positions != nil && m.selectedShow.MALID != "" && m.selectedShow.MALID != "0" {
		if sState, ok := positions[m.selectedShow.MALID]; ok {
			showState = &sState
			maxCompleted := 0.0
			for _, ep := range sState.CompletedEpisodes {
				if ep > maxCompleted {
					maxCompleted = ep
				}
			}
			nextEpVal := maxCompleted + 1.0
			if sState.ResumeState != nil && sState.ResumeState.Episode > maxCompleted {
				nextEpVal = sState.ResumeState.Episode
			}
			nextEp = fmt.Sprintf("%.1f", nextEpVal)
			if strings.HasSuffix(nextEp, ".0") {
				nextEp = nextEp[:len(nextEp)-2]
			}
		}
	}

	if nextEp == "" && lastEp != "" {
		for i, ep := range m.episodes {
			if ep == lastEp {
				if i+1 < len(m.episodes) {
					nextEp = m.episodes[i+1]
				}
				break
			}
		}
	}

	subCount := m.selectedShow.SubCount()
	dubCount := m.selectedShow.DubCount()

	var items []list.Item
	for _, ep := range m.episodes {
		isNext := ep == nextEp
		title := ""
		desc := ""
		if info, ok := m.episodeDetails[ep]; ok {
			title = fmt.Sprintf("Ep %s: %s", ep, info.Title)
			var tags []string
			if info.Filler {
				tags = append(tags, "Filler")
			}
			if info.Recap {
				tags = append(tags, "Recap")
			}
			if info.Aired != "" {
				tags = append(tags, "Aired: "+info.Aired)
			}
			desc = strings.Join(tags, " | ")
		} else {
			title = fmt.Sprintf("Episode %s", ep)
		}

		subAvail := m.selectedShow.HasSub(ep)
		dubAvail := m.selectedShow.HasDub(ep)

		isCompleted := false
		if showState != nil {
			epVal := parseEpisodeNumber(ep)
			for _, completedEp := range showState.CompletedEpisodes {
				if completedEp == epVal {
					isCompleted = true
					break
				}
			}
		}

		items = append(items, episodeItem{
			epNo:        ep,
			isNext:      isNext,
			isCompleted: isCompleted,
			title:       title,
			desc:        desc,
			subAvail:    subAvail,
			dubAvail:    dubAvail,
		})
	}
	m.episodeItems = items
	
	// Add sub/dub badge to the list title
	var badge string
	if subCount > 0 && dubCount > 0 {
		badge = " [SUB+DUB]"
	} else if subCount > 0 {
		badge = " [SUB only]"
	} else if dubCount > 0 {
		badge = " [DUB only]"
	}
	m.episodeList.Title = fmt.Sprintf("Select Episode%s", badge)
	m.episodeList.SetItems(items)
}

func parseEpisodeNumber(ep string) float64 {
	var numStr strings.Builder
	hasDot := false
	for _, r := range ep {
		if r >= '0' && r <= '9' {
			numStr.WriteRune(r)
		} else if r == '.' && !hasDot {
			numStr.WriteRune(r)
			hasDot = true
		} else if numStr.Len() > 0 {
			break
		}
	}
	if numStr.Len() == 0 {
		return 0
	}
	var val float64
	fmt.Sscanf(numStr.String(), "%f", &val)
	return val
}

func tickMpvStatusCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		status, err := queryMpvStatus()
		return tickMpvStatusMsg{status: status, err: err}
	})
}

func doFetchAllStreams(showID, mode, epNo string) tea.Cmd {
	return func() tea.Msg {
		streams, err := fetchAllResolvedStreams(showID, mode, epNo)
		return allStreamsResultMsg{epNo: epNo, streams: streams, err: err}
	}
}

func doFetchShowDetails(showID string) tea.Cmd {
	return func() tea.Msg {
		show, _, err := fetchEpisodeList(showID, "sub")
		return showDetailsResultMsg{showID: showID, show: show, err: err}
	}
}

func doFetchCoverArt(showID, urlStr string, width, height int) tea.Cmd {
	return nil
}

func (m model) renderShowDetailsPanel(show AnimeShow, coverArtANSI string, width, height int) string {
	var s strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#bb9af7")) // Tokyonight purple
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#c0caf5"))
	metaKeyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89"))
	metaValStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff"))
	
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7aa2f7")).
		Padding(1, 2).
		Width(width).
		Height(height)

	// Format metadata lines
	scoreStr := "N/A"
	if show.Score > 0 {
		stars := int(show.Score / 2.0 + 0.5)
		if stars < 1 { stars = 1 }
		if stars > 5 { stars = 5 }
		var starBuf strings.Builder
		for i := 0; i < 5; i++ {
			if i < stars {
				starBuf.WriteString("★")
			} else {
				starBuf.WriteString("☆")
			}
		}
		goldStars := lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68")).Render(starBuf.String())
		scoreStr = fmt.Sprintf("%s  %.2f", goldStars, show.Score)
	}

	seasonStr := "Unknown"
	if show.Season.Quarter != "" && show.Season.Year > 0 {
		seasonStr = fmt.Sprintf("%s %d", show.Season.Quarter, show.Season.Year)
	}

	typeStr := "Unknown"
	if show.Type != "" {
		typeStr = show.Type
	}

	epsStr := "Unknown"
	if show.Duration != "" {
		epsStr = show.Duration
	} else if show.EpCount() > 0 {
		epsStr = fmt.Sprintf("%d Episodes", show.EpCount())
	}

	desc := cleanHTML(show.Description)
	if desc == "" {
		desc = "No synopsis available."
	}
	
	// Layout size calculations:
	rightColWidth := width - 6
	if rightColWidth < 15 {
		rightColWidth = 15
	}
	
	rightBodyStyle := lipgloss.NewStyle().Width(rightColWidth).Foreground(lipgloss.Color("#a9b1d6"))

	// We calculate remaining lines in the container to truncate description gracefully
	// Header is 1 line, title wraps (assume 1-2 lines), separator/margins/spacing is ~6 lines.
	// That's roughly 8 lines of overhead.
	overhead := 13
	descMaxHeight := height - overhead - 2 // padding
	if descMaxHeight < 3 {
		descMaxHeight = 3
	}

	// Truncate description to fit nicely with scroll support
	descLines := strings.Split(rightBodyStyle.Render(desc), "\n")
	maxScroll := len(descLines) - descMaxHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.detailsScrollOffset > maxScroll {
		m.detailsScrollOffset = maxScroll
	}
	if m.detailsScrollOffset < 0 {
		m.detailsScrollOffset = 0
	}

	visibleLines := descLines
	if len(descLines) > descMaxHeight {
		start := m.detailsScrollOffset
		end := start + descMaxHeight
		if end > len(descLines) {
			end = len(descLines)
		}
		visibleLines = descLines[start:end]
	}
	truncatedDesc := strings.Join(visibleLines, "\n")

	synopsisHeader := "◆ SYNOPSIS ◆"
	if maxScroll > 0 {
		currLine := m.detailsScrollOffset + 1
		maxLine := maxScroll + 1
		synopsisHeader = fmt.Sprintf("◆ SYNOPSIS (scroll: h/l) [%d/%d] ◆", currLine, maxLine)
	}

	rightPanelContent := fmt.Sprintf(
		"%s\n\n%s\n%s\n%s\n%s\n\n%s\n%s",
		titleStyle.Render(show.Name),
		fmt.Sprintf("%s %s", metaKeyStyle.Render("Rating:  "), scoreStr),
		fmt.Sprintf("%s %s", metaKeyStyle.Render("Format:  "), metaValStyle.Render(typeStr)),
		fmt.Sprintf("%s %s", metaKeyStyle.Render("Release: "), metaValStyle.Render(seasonStr)),
		fmt.Sprintf("%s %s", metaKeyStyle.Render("Length:  "), metaValStyle.Render(epsStr)),
		headerStyle.Render(synopsisHeader),
		truncatedDesc,
	)

	panelContent := fmt.Sprintf(
		"%s\n%s",
		headerStyle.Render("◆ SHOW DETAILS ◆"),
		rightPanelContent,
	)

	s.WriteString(borderStyle.Render(panelContent))
	return s.String()
}

func (m model) renderEpisodeDetailsPanel(width, height int) string {
	var s strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#bb9af7")) // Tokyonight purple
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#c0caf5"))
	metaKeyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89"))
	metaValStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff"))
	
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7aa2f7")).
		Padding(1, 2).
		Width(width).
		Height(height)

	item := m.episodeList.SelectedItem()
	if item == nil {
		return borderStyle.Render("No episode selected.")
	}

	epItem, ok := item.(episodeItem)
	if !ok {
		return borderStyle.Render("No episode selected.")
	}

	title := fmt.Sprintf("Episode %s", epItem.epNo)
	aired := "Unknown"
	classification := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")).Render("Canon") // green for canon

	synopsis := "No synopsis available."
	if info, ok := m.episodeDetails[epItem.epNo]; ok {
		if info.Title != "" {
			title = fmt.Sprintf("Ep %s: %s", epItem.epNo, info.Title)
		}
		if info.Aired != "" {
			aired = info.Aired
		}
		if info.Synopsis != "" {
			synopsis = cleanHTML(info.Synopsis)
		}
		if info.Filler {
			classification = lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e")).Render("Filler (Non-Canon)")
		} else if info.Recap {
			classification = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68")).Render("Recap")
		}
	} else if m.selectedShow.MALID == "" || m.selectedShow.MALID == "0" {
		title = fmt.Sprintf("Episode %s", epItem.epNo)
		aired = "N/A (No MAL ID)"
		classification = lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Render("Unknown (No MAL)")
	} else {
		title = fmt.Sprintf("Episode %s", epItem.epNo)
		aired = "Loading..."
		classification = lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Render("Loading...")
	}

		// Layout size calculations:
	rightColWidth := width - 6
	if rightColWidth < 15 {
		rightColWidth = 15
	}

	// Calculate multiplexing status flags
	videoStatus := "Video: RESOLVED"
	audioStatus := "Audio: SINGLE-STREAM"
	if m.mode == "dual" {
		audioStatus = "Audio: DUAL-MAPPED"
	}
	metadataFlags := fmt.Sprintf("%s  •  %s", videoStatus, audioStatus)

	// Calculate AniSkip pre-flight badge
	aniSkipBadge := "✨ AniSkip Checking..."
	if ready, checked := m.aniSkipReady[epItem.epNo]; checked {
		if ready {
			aniSkipBadge = "✨ AniSkip Ready"
		} else {
			aniSkipBadge = "✨ AniSkip Unavailable"
		}
	}

	// Calculate visual inline progress bar
	progressBar := ""
	positions, _ := loadPositions()
	if positions != nil && m.selectedShow.MALID != "" {
		if showState, ok := positions[m.selectedShow.MALID]; ok && showState.ResumeState != nil {
			reqEp := parseEpisodeNumber(epItem.epNo)
			if showState.ResumeState.Episode == reqEp {
				pos := showState.ResumeState.PositionSeconds
				total := showState.ResumeState.TotalSeconds
				if total > 0 {
					pct := pos / total
					if pct > 1.0 { pct = 1.0 }
					if pct < 0.0 { pct = 0.0 }
					progressBar = fmt.Sprintf("[%s] %d%% watched", renderSmoothProgressBar(pct, 12), int(pct*100))
				}
			}
		}
	}

	metaLines := []string{
		fmt.Sprintf("%s %s", metaKeyStyle.Render("Release Date:  "), metaValStyle.Render(aired)),
		fmt.Sprintf("%s %s", metaKeyStyle.Render("Classification:"), classification),
		fmt.Sprintf("%s %s", metaKeyStyle.Render("Format/Audio:  "), metaValStyle.Render(metadataFlags)),
		fmt.Sprintf("%s %s", metaKeyStyle.Render("AniSkip:       "), lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff")).Render(aniSkipBadge)),
	}
	if progressBar != "" {
		metaLines = append(metaLines, fmt.Sprintf("%s %s", metaKeyStyle.Render("Progress:      "), lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")).Render(progressBar)))
	}

	// Calculate spacing and height for synopsis
	overhead := len(metaLines) + 8
	synMaxHeight := height - overhead
	if synMaxHeight < 3 {
		synMaxHeight = 3
	}

	synBodyStyle := lipgloss.NewStyle().Width(rightColWidth).Foreground(lipgloss.Color("#a9b1d6"))
	synLines := strings.Split(synBodyStyle.Render(synopsis), "\n")
	maxScroll := len(synLines) - synMaxHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.detailsScrollOffset > maxScroll {
		m.detailsScrollOffset = maxScroll
	}
	if m.detailsScrollOffset < 0 {
		m.detailsScrollOffset = 0
	}

	visibleLines := synLines
	if len(synLines) > synMaxHeight {
		start := m.detailsScrollOffset
		end := start + synMaxHeight
		if end > len(synLines) {
			end = len(synLines)
		}
		visibleLines = synLines[start:end]
	}
	wrappedSynopsis := strings.Join(visibleLines, "\n")

	synopsisHeader := "◆ EPISODE SYNOPSIS ◆"
	if maxScroll > 0 {
		currLine := m.detailsScrollOffset + 1
		maxLine := maxScroll + 1
		synopsisHeader = fmt.Sprintf("◆ EPISODE SYNOPSIS (scroll: h/l) [%d/%d] ◆", currLine, maxLine)
	}

	rightPanelContent := fmt.Sprintf(
		"%s\n%s\n\n%s\n\n%s\n%s",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Render(m.selectedShow.Name),
		titleStyle.Render(title),
		strings.Join(metaLines, "\n"),
		headerStyle.Render(synopsisHeader),
		wrappedSynopsis,
	)

	panelContent := fmt.Sprintf(
		"%s\n%s",
		headerStyle.Render("◆ EPISODE DETAILS ◆"),
		rightPanelContent,
	)

	s.WriteString(borderStyle.Render(panelContent))
	return s.String()
}

func cleanHTML(input string) string {
	s := html.UnescapeString(input)
	reBr := regexp.MustCompile(`(?i)<br\s*/?>`)
	s = reBr.ReplaceAllString(s, "\n")
	reTags := regexp.MustCompile(`<[^>]*>`)
	s = reTags.ReplaceAllString(s, "")
	
	// Normalize typographic punctuation to standard ASCII
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "‘", "'")
	s = strings.ReplaceAll(s, "“", "\"")
	s = strings.ReplaceAll(s, "”", "\"")
	s = strings.ReplaceAll(s, "–", "-")
	s = strings.ReplaceAll(s, "—", "-")
	s = strings.ReplaceAll(s, "…", "...")
	return strings.TrimSpace(s)
}

func nextMode(current string) string {
	switch current {
	case "dual":
		return "sub"
	case "sub":
		return "dub"
	default:
		return "dual"
	}
}

func (m *model) dynamicListHeight() int {
	offset := 6
	if m.state == stateHistory || m.state == stateSearchInput || m.state == stateSearchRunning || m.state == stateShowSelect || m.state == stateEpisodeSelect || m.state == stateSourceSelect || m.state == stateLogs || m.state == stateConfig {
		offset = 9
	}
	if m.playbackActive {
		offset += 5 // status bar is 4 lines height + 1 line spacing border
	}
	h := m.height - offset
	if h < 5 {
		return 5
	}
	return h
}

func (m *model) recalculateSizes() {
	leftWidth := m.width - 4
	if m.width >= 80 {
		leftWidth = m.width / 2
		if leftWidth < 35 {
			leftWidth = 35
		}
	}
	
	listHeight := m.dynamicListHeight()
	m.historyList.SetSize(leftWidth, listHeight)
	m.showList.SetSize(leftWidth, listHeight)
	m.episodeList.SetSize(leftWidth, listHeight)
	m.sourceList.SetSize(leftWidth, listHeight)
	
	m.telemetryViewport.Width = m.width - 4
	m.telemetryViewport.Height = listHeight
}

func (m model) getConfig() Config {
	cfg := loadConfig()
	cfg.Autoplay = m.autoplay
	cfg.Autoskip = m.autoskip
	cfg.SkipFillers = m.skipFillers
	cfg.PreferredMode = m.mode
	cfg.PreferredQuality = m.quality
	return cfg
}

func formatBool(b bool) string {
	if b {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9ece6a")).Render("[ON]")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e")).Render("[OFF]")
}

func formatLogLine(line string) string {
	lower := strings.ToLower(line)
	if strings.Contains(line, "[ERROR]") || strings.Contains(lower, "error") || strings.Contains(lower, "failed") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e")).Render(line)
	}
	if strings.Contains(line, "[WARN]") || strings.Contains(lower, "warning") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68")).Render(line)
	}
	if strings.Contains(line, "[API]") || strings.Contains(line, "http request") || strings.Contains(line, "http response") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff")).Render(line)
	}
	if strings.Contains(line, "[MPV]") {
		parts := strings.SplitN(line, "[MPV]", 2)
		prefix := parts[0]
		content := parts[1]
		mpvBadge := lipgloss.NewStyle().Foreground(lipgloss.Color("#bb9af7")).Render("[MPV]")
		return prefix + mpvBadge + lipgloss.NewStyle().Foreground(lipgloss.Color("#a9b1d6")).Render(content)
	}
	if strings.Contains(line, "[INFO]") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff")).Render(line)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#c0caf5")).Render(line)
}
