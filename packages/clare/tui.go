package main

import (
	"fmt"
	"os"
	"os/exec"
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
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#8A2BE2")). // Beautiful violet
			Padding(0, 2).
			MarginBottom(1)

	accentColorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#EE6FF8"))

	cyanColorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00F0FF"))

	grayColorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF3366"))

	normalTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D3D3D3"))

	selectedTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#EE6FF8"))
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
func (s showItem) Description() string { return fmt.Sprintf("%d episodes available", s.show.EpCount()) }
func (s showItem) FilterValue() string { return s.show.Name }

type episodeItem struct {
	epNo   string
	isNext bool
}

func (e episodeItem) Title() string {
	if e.isNext {
		return fmt.Sprintf("Episode %s (Next Up)", e.epNo)
	}
	return fmt.Sprintf("Episode %s", e.epNo)
}
func (e episodeItem) Description() string { return "" }
func (e episodeItem) FilterValue() string { return e.epNo }

// Msg definitions
type searchResultMsg struct {
	shows []AnimeShow
	err   error
}

type episodesResultMsg struct {
	episodes []string
	err      error
}

type resolvedPlaybackMsg struct {
	cmd         *exec.Cmd
	tempLuaFile string
	err         error
}

type playbackFinishedMsg struct {
	err error
}

// Bubble Tea Model
type model struct {
	state          tuiState
	historyItems   []list.Item
	historyList    list.Model
	searchInput    textinput.Model
	spinner        spinner.Model
	showItems      []list.Item
	showList       list.Model
	episodeItems   []list.Item
	episodeList    list.Model
	selectedShow   AnimeShow
	selectedEp     string
	episodes       []string
	download       bool
	quality        string
	mode           string // sub, dub, dual
	err            error
	width, height  int
	loadingMsg     string
	tempLuaFile    string
	initialSearch  string
}

func initialModel(initialSearch, mode, quality string, download bool) model {
	// Setup spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#EE6FF8"))

	// Setup searchInput
	ti := textinput.New()
	ti.Placeholder = "Enter anime title..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 30

	// Setup lists with placeholder items
	hList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	hList.Title = "Continue Watching"
	hList.SetShowStatusBar(false)

	sList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	sList.Title = "Search Results"
	sList.SetShowStatusBar(false)

	eList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	eList.Title = "Select Episode"
	eList.SetShowStatusBar(false)

	m := model{
		state:         stateHistory,
		historyList:   hList,
		searchInput:   ti,
		spinner:       s,
		showList:      sList,
		episodeList:   eList,
		mode:          mode,
		quality:       quality,
		download:      download,
		initialSearch: initialSearch,
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

		m.episodes = msg.episodes
		m.state = stateEpisodeSelect

		// Determine if there is a "next episode" to suggest based on watch history
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
		selectIndex := 0
		for i, ep := range m.episodes {
			isNext := ep == nextEp
			if isNext {
				selectIndex = i
			}
			items = append(items, episodeItem{epNo: ep, isNext: isNext})
		}
		m.episodeItems = items
		m.episodeList.SetItems(items)
		m.episodeList.Select(selectIndex)
		return m, nil

	case resolvedPlaybackMsg:
		debugLog("TUI resolvedPlaybackMsg: err=%v, tempLuaFile=%s", msg.err, msg.tempLuaFile)
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
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
		}

		// Re-fetch episode list to update the "Next Up" indicator and highlights
		m.state = stateSearchRunning
		m.loadingMsg = "Refreshing episode list..."
		return m, doFetchEpisodes(m.selectedShow.ID, "sub")

	case tea.KeyMsg:
		debugLog("TUI KeyMsg: key=%s, state=%d", msg.String(), m.state)
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
	s.WriteString(titleStyle.Render(" CLARE ANIME CLIENT "))
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
		s.WriteString(accentColorStyle.Render(fmt.Sprintf("Show: %s", m.selectedShow.Name)) + "\n\n")
		s.WriteString(m.episodeList.View())
		s.WriteString("\n\n" + helpStyle("enter: play episode | esc: back | q: quit"))

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
		eps, err := fetchEpisodeList(showID, mode)
		return episodesResultMsg{episodes: eps, err: err}
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
				cmd, tempLua, err = playSingleCmd(dubStream, selectedShow.Name, epNo)
			} else if errDub != nil {
				cmd, tempLua, err = playSingleCmd(subStream, selectedShow.Name, epNo)
			} else {
				subTracks := countAudioStreams(subStream)
				cmd, tempLua, err = playDualCmd(subStream, dubStream, subTracks, selectedShow.Name, epNo)
			}
		} else if mode == "dub" {
			dubStream, errDub := resolveStreamURL(selectedShow.ID, "dub", epNo, quality)
			if errDub != nil {
				return resolvedPlaybackMsg{err: errDub}
			}
			cmd, tempLua, err = playSingleCmd(dubStream, selectedShow.Name, epNo)
		} else {
			subStream, errSub := resolveStreamURL(selectedShow.ID, "sub", epNo, quality)
			if errSub != nil {
				return resolvedPlaybackMsg{err: errSub}
			}
			cmd, tempLua, err = playSingleCmd(subStream, selectedShow.Name, epNo)
		}

		return resolvedPlaybackMsg{cmd: cmd, tempLuaFile: tempLua, err: err}
	}
}
