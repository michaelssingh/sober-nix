package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
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
	statePlaybackPreparing
	statePlaybackActive
	stateError
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
)

// List items definitions

type historyItem struct {
	showID    string
	showName  string
	lastEp    string
	timestamp int64
}

func (h historyItem) Title() string       { return h.showName }
func (h historyItem) Description() string {
	t := time.Unix(h.timestamp, 0).Format("2006-01-02 15:04")
	return fmt.Sprintf("Last watched: Ep %s | %s", h.lastEp, t)
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
	return strings.Join(parts, "  •  ")
}
func (s showItem) FilterValue() string { return s.show.Name }

type episodeItem struct {
	epNo   string
	isNext bool
	title  string
	desc   string
}

func (e episodeItem) Title() string {
	if e.title != "" {
		if e.isNext {
			return fmt.Sprintf("%s (Next Up)", e.title)
		}
		return e.title
	}
	if e.isNext {
		return fmt.Sprintf("Episode %s (Next Up)", e.epNo)
	}
	return fmt.Sprintf("Episode %s", e.epNo)
}
func (e episodeItem) Description() string { return e.desc }
func (e episodeItem) FilterValue() string { return e.epNo }

type JikanEpInfo struct {
	Title  string
	Aired  string
	Filler bool
	Recap  bool
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

type coverArtResultMsg struct {
	showID string
	ansi   string
}

type resolvedPlaybackMsg struct {
	warning string
	cmd         *exec.Cmd
	tempLuaFile string
	err         error
}

type playbackFinishedMsg struct {
	err error
}

// Bubble Tea Model
type model struct {
	state              tuiState
	historyItems       []list.Item
	historyList        list.Model
	searchInput        textinput.Model
	spinner            spinner.Model
	showItems          []list.Item
	showList           list.Model
	episodeItems       []list.Item
	episodeList        list.Model
	selectedShow       AnimeShow
	selectedEp         string
	episodes           []string
	download           bool
	quality            string
	mode               string // sub, dub, dual
	err                error
	width, height      int
	loadingMsg         string
	tempLuaFile        string
	initialSearch      string
	episodeDetails     map[string]JikanEpInfo
	loadedJikanPages   map[int]bool
	autoplay           bool
	triggerAutoplay    bool
	historyShowDetails map[string]AnimeShow
	coverArtCache      map[string]string
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

	hList := createMinimalList("Continue Watching")
	sList := createMinimalList("Search Results")
	eList := createMinimalList("Select Episode")

	m := model{
		state:              stateHistory,
		historyList:        hList,
		searchInput:        ti,
		spinner:            s,
		showList:           sList,
		episodeList:        eList,
		mode:               mode,
		quality:            quality,
		download:           download,
		initialSearch:      initialSearch,
		episodeDetails:     make(map[string]JikanEpInfo),
		loadedJikanPages:   make(map[int]bool),
		autoplay:           true, // Autoplay on by default
		historyShowDetails: make(map[string]AnimeShow),
		coverArtCache:      make(map[string]string),
	}

	m.refreshHistory()

	if initialSearch != "" {
		m.state = stateSearchRunning
		m.loadingMsg = fmt.Sprintf("Searching for %q...", initialSearch)
	} else if len(m.historyItems) == 0 {
		m.state = stateSearchInput
	}

	return m
}

func (m *model) refreshHistory() {
	rawHist, err := loadHistory()
	if err == nil {
		uniq := getUniqueHistory(rawHist)
		var items []list.Item
		for _, u := range uniq {
			items = append(items, historyItem{
				showID:    u.ShowID,
				showName:  u.ShowName,
				lastEp:    u.Episode,
				timestamp: u.Timestamp,
			})
		}
		m.historyItems = items
		m.historyList.SetItems(items)
	}
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, m.spinner.Tick)
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
		
		// Set sizes for all lists
		listHeight := m.height - 6
		if listHeight < 5 {
			listHeight = 5
		}
		leftWidth := m.width - 4
		if m.width >= 80 {
			leftWidth = m.width / 2
			if leftWidth < 35 {
				leftWidth = 35
			}
		}
		m.historyList.SetSize(leftWidth, listHeight)
		m.showList.SetSize(leftWidth, listHeight)
		m.episodeList.SetSize(leftWidth, listHeight)
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

		// If autoplay was triggered, fetch next stream immediately
		if m.triggerAutoplay {
			m.triggerAutoplay = false
			var nextEpNo string
			foundNext := false
			for _, item := range m.episodeItems {
				if epItem, ok := item.(episodeItem); ok && epItem.isNext {
					nextEpNo = epItem.epNo
					foundNext = true
					break
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
		debugLog("TUI resolvedPlaybackMsg: err=%v, warning=%s, tempLuaFile=%s", msg.err, msg.warning, msg.tempLuaFile)
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		if msg.warning != "" {
			m.loadingMsg = msg.warning
		}

		m.state = statePlaybackActive
		m.tempLuaFile = msg.tempLuaFile

		// Suspend TUI and run player command
		return m, tea.ExecProcess(msg.cmd, func(err error) tea.Msg {
			return playbackFinishedMsg{err}
		})

	case playbackFinishedMsg:
		debugLog("TUI playbackFinishedMsg: err=%v", msg.err)
		// Delete temporary Lua script
		if m.tempLuaFile != "" {
			_ = os.Remove(m.tempLuaFile)
			m.tempLuaFile = ""
		}

		// Update watch history if no playback launch error
		if msg.err == nil {
			_ = recordWatch(m.selectedShow.ID, m.selectedShow.Name, m.selectedEp)
			m.refreshHistory()
			if m.autoplay {
				m.triggerAutoplay = true
			}
		}

		// Re-fetch episode list to update the "Next Up" indicator and highlights
		m.state = stateSearchRunning
		m.loadingMsg = "Refreshing episode list..."
		return m, doFetchEpisodes(m.selectedShow.ID, "sub")

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

	case coverArtResultMsg:
		if msg.ansi != "" {
			m.coverArtCache[msg.showID] = msg.ansi
		} else {
			m.coverArtCache[msg.showID] = ""
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
		return m, nil

	case tea.KeyMsg:
		debugLog("TUI KeyMsg: key=%s, state=%d", msg.String(), m.state)

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
		case "ctrl+c", "q":
			// Don't quit with 'q' if we are typing in text input
			if m.state != stateSearchInput {
				return m, tea.Quit
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
			case "s", "/":
				m.state = stateSearchInput
				m.searchInput.Reset()
				m.searchInput.Focus()
				return m, nil
			}
			var cmd tea.Cmd
			m.historyList, cmd = m.historyList.Update(msg)
			
			// Trigger details fetch for currently highlighted history item if not loaded
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
					return m, tea.Batch(append(cmds, cmd)...)
				}
			}
			return m, cmd

		case stateSearchInput:
			switch msg.String() {
			case "enter":
				query := strings.TrimSpace(m.searchInput.Value())
				if query != "" {
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
			}
			var cmd tea.Cmd
			m.showList, cmd = m.showList.Update(msg)
			if selected, ok := m.showList.SelectedItem().(showItem); ok {
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
					m.state = statePlaybackPreparing
					m.loadingMsg = fmt.Sprintf("Preparing playback for Episode %s...", selected.epNo)
					return m, doPreparePlayback(m.selectedShow, selected.epNo, m.mode, m.quality, m.download)
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

			// Lazy load metadata page for currently selected episode if needed
			if item := m.episodeList.SelectedItem(); item != nil {
				if epItem, ok := item.(episodeItem); ok {
					var val int
					fmt.Sscanf(epItem.epNo, "%d", &val)
					if val > 0 {
						page := (val - 1) / 100 + 1
						if !m.loadedJikanPages[page] {
							m.loadedJikanPages[page] = true
							cmds = append(cmds, doFetchJikanMetadata(m.selectedShow.MALID, page))
						}
					}
				}
			}
			return m, tea.Batch(cmds...)

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
	s.WriteString(titleStyle.Render(" CLARE "))
	s.WriteString("\n")

	listHeight := m.height - 6
	if listHeight < 5 {
		listHeight = 5
	}

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
					rightView = renderShowDetailsPanel(show, art, rightWidth, listHeight)
				} else {
					tempShow := AnimeShow{ID: selected.showID, Name: selected.showName, Description: "Loading details..."}
					rightView = renderShowDetailsPanel(tempShow, art, rightWidth, listHeight)
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
		s.WriteString("\n\n" + helpStyle("s: search | enter: select show | q: quit"))

	case stateSearchInput:
		s.WriteString(accentColorStyle.Render("Search Anime:") + "\n\n")
		s.WriteString(m.searchInput.View())
		s.WriteString("\n\n" + helpStyle("enter: search | esc: cancel"))

	case stateSearchRunning:
		s.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), m.loadingMsg))

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
				rightView = renderShowDetailsPanel(selected.show, art, rightWidth, listHeight)
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

	case statePlaybackPreparing:
		s.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), m.loadingMsg))

	case statePlaybackActive:
		s.WriteString(cyanColorStyle.Render("Playback active in mpv. Controlling stream...\n"))

	case stateError:
		s.WriteString(errorStyle.Render("Error encountered:") + "\n\n")
		s.WriteString(fmt.Sprintf("  %v\n\n", m.err))
		s.WriteString(helpStyle("press enter or esc to return to search"))
	}

	return s.String()
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
				cmd, tempLua, err = playSingleCmd(dubStream, selectedShow.Name, epNo, selectedShow.MALID)
			} else if errDub != nil {
				debugLog("doPreparePlayback: dub failed (%v), falling back to sub-only", errDub)
				cmd, tempLua, err = playSingleCmd(subStream, selectedShow.Name, epNo, selectedShow.MALID)
				if err == nil {
					return resolvedPlaybackMsg{cmd: cmd, tempLuaFile: tempLua, warning: fmt.Sprintf("⚠ Dub unavailable (%v) — playing sub only", errDub)}
				}
			} else {
				debugLog("doPreparePlayback: both streams resolved, launching dual-audio")
				cmd, tempLua, err = playDualCmd(subStream, dubStream, selectedShow.Name, epNo, selectedShow.MALID)
			}
		} else if mode == "dub" {
			dubStream, errDub := resolveStreamURL(selectedShow.ID, "dub", epNo, quality)
			if errDub != nil {
				return resolvedPlaybackMsg{err: errDub}
			}
			cmd, tempLua, err = playSingleCmd(dubStream, selectedShow.Name, epNo, selectedShow.MALID)
		} else {
			subStream, errSub := resolveStreamURL(selectedShow.ID, "sub", epNo, quality)
			if errSub != nil {
				return resolvedPlaybackMsg{err: errSub}
			}
			cmd, tempLua, err = playSingleCmd(subStream, selectedShow.Name, epNo, selectedShow.MALID)
		}

		return resolvedPlaybackMsg{cmd: cmd, tempLuaFile: tempLua, err: err}
	}
}


func doFetchJikanMetadata(malID string, page int) tea.Cmd {
	return func() tea.Msg {
		if malID == "" || malID == "0" {
			return jikanMetadataMsg{malID: malID, page: page, err: fmt.Errorf("no MAL ID")}
		}
		client := &http.Client{Timeout: 8 * time.Second}
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
				MalID  int    `json:"mal_id"`
				Title  string `json:"title"`
				Aired  string `json:"aired"`
				Filler bool   `json:"filler"`
				Recap  bool   `json:"recap"`
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
				Title:  d.Title,
				Aired:  airedStr,
				Filler: d.Filler,
				Recap:  d.Recap,
			}
		}
		return jikanMetadataMsg{malID: malID, page: page, metadata: metadata}
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
	if lastEp != "" {
		for i, ep := range m.episodes {
			if ep == lastEp {
				if i+1 < len(m.episodes) {
					nextEp = m.episodes[i+1]
				}
				break
			}
		}
	}

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

		items = append(items, episodeItem{
			epNo:   ep,
			isNext: isNext,
			title:  title,
			desc:   desc,
		})
	}
	m.episodeItems = items
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

func doFetchShowDetails(showID string) tea.Cmd {
	return func() tea.Msg {
		show, _, err := fetchEpisodeList(showID, "sub")
		return showDetailsResultMsg{showID: showID, show: show, err: err}
	}
}

func doFetchCoverArt(showID, urlStr string, width, height int) tea.Cmd {
	return func() tea.Msg {
		if urlStr == "" {
			return coverArtResultMsg{showID: showID, ansi: ""}
		}
		imgPath, err := downloadThumbnail(showID, urlStr)
		if err != nil {
			debugLog("doFetchCoverArt download failed: %v", err)
			return coverArtResultMsg{showID: showID, ansi: ""}
		}
		ansi := renderImageANSI(imgPath, width, height)
		return coverArtResultMsg{showID: showID, ansi: ansi}
	}
}

func renderShowDetailsPanel(show AnimeShow, coverArtANSI string, width, height int) string {
	var s strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#bb9af7")) // Tokyonight purple
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#c0caf5"))
	metaKeyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89"))
	metaValStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff"))
	
	// Full-height border aligned with the list
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
	if show.EpCount() > 0 {
		epsStr = fmt.Sprintf("%d Episodes", show.EpCount())
	}

	desc := cleanHTML(show.Description)
	if desc == "" {
		desc = "No synopsis available."
	}

	// Layout size calculations:
	// We split horizontally: Left is cover art, Right is metadata/synopsis
	imgWidth := 16
	imgHeight := 11
	
	var leftPanel string
	if coverArtANSI == "Loading..." {
		placeholderStyle := lipgloss.NewStyle().
			Width(imgWidth).
			Height(imgHeight).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3b4261")).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("#565f89")).
			Background(lipgloss.Color("#1a1b26"))
		leftPanel = placeholderStyle.Render("Loading\nArt...")
	} else if coverArtANSI != "" {
		frameStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#3b4261")).
			Padding(0)
		if strings.HasPrefix(coverArtANSI, "\x1b") {
			emptySpaces := ""
			for i := 0; i < imgHeight; i++ {
				emptySpaces += strings.Repeat(" ", imgWidth)
				if i < imgHeight-1 {
					emptySpaces += "\n"
				}
			}
			leftPanel = frameStyle.Render(emptySpaces)
			leftPanel = "\x1b[s\x1b[1C\x1b[1B" + coverArtANSI + "\x1b[u" + leftPanel
		} else {
			leftPanel = frameStyle.Render(coverArtANSI)
		}
	} else {
		placeholderStyle := lipgloss.NewStyle().
			Width(imgWidth).
			Height(imgHeight).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3b4261")).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("#565f89")).
			Background(lipgloss.Color("#1a1b26"))
		leftPanel = placeholderStyle.Render("No Cover\nArt")
	}

	rightColWidth := width - imgWidth - 8
	if rightColWidth < 15 {
		rightColWidth = 15
	}
	
	rightBodyStyle := lipgloss.NewStyle().Width(rightColWidth).Foreground(lipgloss.Color("#a9b1d6"))

	// We calculate remaining lines in the container to truncate description gracefully
	// Header is 1 line, title wraps (assume 1-2 lines), separator/margins/spacing is ~6 lines.
	// That's roughly 8 lines of overhead.
	overhead := 8
	descMaxHeight := height - overhead - 2 // padding
	if descMaxHeight < 3 {
		descMaxHeight = 3
	}

	// Truncate description to fit nicely
	descLines := strings.Split(rightBodyStyle.Render(desc), "\n")
	if len(descLines) > descMaxHeight {
		descLines = descLines[:descMaxHeight]
		lastLine := descLines[len(descLines)-1]
		if len(lastLine) > 3 {
			descLines[len(descLines)-1] = lastLine[:len(lastLine)-3] + "..."
		} else {
			descLines[len(descLines)-1] = "..."
		}
	}
	truncatedDesc := strings.Join(descLines, "\n")

	rightPanelContent := fmt.Sprintf(
		"%s\n\n%s\n%s\n%s\n%s\n\n%s\n%s",
		titleStyle.Render(show.Name),
		fmt.Sprintf("%s %s", metaKeyStyle.Render("Rating:  "), scoreStr),
		fmt.Sprintf("%s %s", metaKeyStyle.Render("Format:  "), metaValStyle.Render(typeStr)),
		fmt.Sprintf("%s %s", metaKeyStyle.Render("Release: "), metaValStyle.Render(seasonStr)),
		fmt.Sprintf("%s %s", metaKeyStyle.Render("Length:  "), metaValStyle.Render(epsStr)),
		headerStyle.Render("◆ SYNOPSIS ◆"),
		truncatedDesc,
	)

	// Combine left and right panels side-by-side
	bodyContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanelContent)

	panelContent := fmt.Sprintf(
		"%s\n%s",
		headerStyle.Render("◆ SHOW DETAILS ◆"),
		bodyContent,
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

	if info, ok := m.episodeDetails[epItem.epNo]; ok {
		if info.Title != "" {
			title = fmt.Sprintf("Ep %s: %s", epItem.epNo, info.Title)
		}
		if info.Aired != "" {
			aired = info.Aired
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
	imgWidth := 16
	imgHeight := 11
	
	coverArtANSI := m.coverArtCache[m.selectedShow.ID]
	var leftPanel string
	if coverArtANSI == "Loading..." {
		placeholderStyle := lipgloss.NewStyle().
			Width(imgWidth).
			Height(imgHeight).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3b4261")).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("#565f89")).
			Background(lipgloss.Color("#1a1b26"))
		leftPanel = placeholderStyle.Render("Loading\nArt...")
	} else if coverArtANSI != "" {
		frameStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#3b4261")).
			Padding(0)
		if strings.HasPrefix(coverArtANSI, "\x1b") {
			emptySpaces := ""
			for i := 0; i < imgHeight; i++ {
				emptySpaces += strings.Repeat(" ", imgWidth)
				if i < imgHeight-1 {
					emptySpaces += "\n"
				}
			}
			leftPanel = frameStyle.Render(emptySpaces)
			leftPanel = "\x1b[s\x1b[1C\x1b[1B" + coverArtANSI + "\x1b[u" + leftPanel
		} else {
			leftPanel = frameStyle.Render(coverArtANSI)
		}
	} else {
		placeholderStyle := lipgloss.NewStyle().
			Width(imgWidth).
			Height(imgHeight).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3b4261")).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("#565f89")).
			Background(lipgloss.Color("#1a1b26"))
		leftPanel = placeholderStyle.Render("No Cover\nArt")
	}

	rightColWidth := width - imgWidth - 8
	if rightColWidth < 15 {
		rightColWidth = 15
	}

	// Calculate spacing for controls at the bottom
	hintsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Width(rightColWidth)
	hints := "\n\n\n\n"
	hintSpaces := height - imgHeight - 11 // spacing
	if hintSpaces > 0 {
		hints = strings.Repeat("\n", hintSpaces)
	}
	hints += fmt.Sprintf(
		"◆ CONTROLS ◆\n%s",
		hintsStyle.Render(fmt.Sprintf("• enter : play episode\n• a     : toggle autoplay\n• m     : toggle mode (current: %s)\n• esc   : back to shows", strings.ToUpper(m.mode))),
	)

	rightPanelContent := fmt.Sprintf(
		"%s\n%s\n\n%s\n%s\n%s",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Render(m.selectedShow.Name),
		titleStyle.Render(title),
		fmt.Sprintf("%s %s", metaKeyStyle.Render("Release Date:  "), metaValStyle.Render(aired)),
		fmt.Sprintf("%s %s", metaKeyStyle.Render("Classification:"), classification),
		hints,
	)

	bodyContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanelContent)

	panelContent := fmt.Sprintf(
		"%s\n%s",
		headerStyle.Render("◆ EPISODE DETAILS ◆"),
		bodyContent,
	)

	s.WriteString(borderStyle.Render(panelContent))
	return s.String()
}

func cleanHTML(input string) string {
	r := strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n")
	s := r.Replace(input)
	re := regexp.MustCompile("<[^>]*>")
	s = re.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
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
