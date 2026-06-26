package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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
	state            tuiState
	historyItems     []list.Item
	historyList      list.Model
	searchInput      textinput.Model
	spinner          spinner.Model
	showItems        []list.Item
	showList         list.Model
	episodeItems     []list.Item
	episodeList      list.Model
	selectedShow     AnimeShow
	selectedEp       string
	episodes         []string
	download         bool
	quality          string
	mode             string // sub, dub, dual
	err              error
	width, height    int
	loadingMsg       string
	tempLuaFile      string
	initialSearch    string
	episodeDetails   map[string]JikanEpInfo
	loadedJikanPages map[int]bool
	autoplay         bool
	triggerAutoplay  bool
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
		state:            stateHistory,
		historyList:      hList,
		searchInput:      ti,
		spinner:          s,
		showList:         sList,
		episodeList:      eList,
		mode:             mode,
		quality:          quality,
		download:         download,
		initialSearch:    initialSearch,
		episodeDetails:   make(map[string]JikanEpInfo),
		loadedJikanPages: make(map[int]bool),
		autoplay:         true, // Autoplay on by default
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
		m.historyList.SetSize(m.width-4, listHeight)
		m.showList.SetSize(m.width-4, listHeight)
		m.episodeList.SetSize(m.width-4, listHeight)
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
			return m, cmd

		case stateEpisodeSelect:
			switch msg.String() {
			case "a":
				m.autoplay = !m.autoplay
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
				} else if len(m.historyItems) > 0 {
					m.state = stateHistory
				} else {
					m.state = stateSearchInput
					m.searchInput.Focus()
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.episodeList, cmd = m.episodeList.Update(msg)

			// Lazy load metadata page for currently selected episode if needed
			if item := m.episodeList.SelectedItem(); item != nil {
				if epItem, ok := item.(episodeItem); ok {
					var val int
					fmt.Sscanf(epItem.epNo, "%d", &val)
					if val > 0 {
						page := (val - 1) / 100 + 1
						if !m.loadedJikanPages[page] {
							m.loadedJikanPages[page] = true
							return m, tea.Batch(cmd, doFetchJikanMetadata(m.selectedShow.MALID, page))
						}
					}
				}
			}
			return m, cmd

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

	switch m.state {
	case stateHistory:
		s.WriteString(m.historyList.View())
		s.WriteString("\n\n" + helpStyle("s: search | enter: select show | q: quit"))

	case stateSearchInput:
		s.WriteString(accentColorStyle.Render("Search Anime:") + "\n\n")
		s.WriteString(m.searchInput.View())
		s.WriteString("\n\n" + helpStyle("enter: search | esc: cancel"))

	case stateSearchRunning:
		s.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), m.loadingMsg))

	case stateShowSelect:
		s.WriteString(m.showList.View())
		s.WriteString("\n\n" + helpStyle("enter: select show | esc: back | q: quit"))

	case stateEpisodeSelect:
		s.WriteString(m.episodeList.View())
		autoplayStr := "autoplay: OFF"
		if m.autoplay {
			autoplayStr = "autoplay: ON"
		}
		s.WriteString("\n\n" + helpStyle(fmt.Sprintf("enter: play | a: toggle autoplay (%s) | esc: back | q: quit", autoplayStr)))

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
