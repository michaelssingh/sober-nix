package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
)

func (m *model) dynamicListHeight() int {
	baseOffset := 4
	if m.playbackActive {
		baseOffset += 5
	}
	h := m.height - baseOffset
	if h < 5 {
		return 5
	}
	return h
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
	key := m.selectedShow.MALID
	if key == "" || key == "0" {
		key = m.selectedShow.ID
	}
	if positions != nil && key != "" {
		if sState, ok := positions[key]; ok {
			showState = &sState
			maxCompleted := 0.0
			for _, ep := range sState.CompletedEpisodes {
				if ep > maxCompleted {
					maxCompleted = ep
				}
			}
			nextEpVal := maxCompleted + 1.0
			if sState.ResumeState != nil {
				nextEpVal = sState.ResumeState.Episode
			}
			for {
				isCompleted := false
				for _, cEp := range sState.CompletedEpisodes {
					if cEp == nextEpVal {
						isCompleted = true
						break
					}
				}
				if !isCompleted {
					break
				}
				nextEpVal += 1.0
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

	today := time.Now().Format("2006-01-02")
	maxReleasedEp := 0.0
	for _, ep := range m.episodes {
		if info, ok := m.episodeDetails[ep]; ok && info.Aired != "" && info.Aired <= today {
			if num := parseEpisodeNumber(ep); num > maxReleasedEp {
				maxReleasedEp = num
			}
		}
	}
	if maxReleasedEp == 0 {
		if m.mode == "dub" && dubCount > 0 {
			maxReleasedEp = float64(dubCount)
		} else if subCount > 0 {
			maxReleasedEp = float64(subCount)
		}
	}

	var items []list.Item
	for _, ep := range m.episodes {
		epNum := parseEpisodeNumber(ep)

		// Omit episodes that have not aired yet for currently airing shows
		if info, ok := m.episodeDetails[ep]; ok && info.Aired != "" && info.Aired > today {
			continue
		}
		if maxReleasedEp > 0 && epNum > maxReleasedEp {
			if info, ok := m.episodeDetails[ep]; !ok || info.Aired == "" || info.Aired > today {
				continue
			}
		}
		isNext := ep == nextEp
		title := ""
		desc := ""
		isMovieMedia := strings.EqualFold(m.selectedShow.Type, "MOVIE") ||
			strings.HasPrefix(m.selectedShow.ID, "vidsrc:movie")

		if isMovieMedia && ep == "1" {
			title = "Full Movie"
		} else if info, ok := m.episodeDetails[ep]; ok {
			if strings.HasPrefix(strings.ToUpper(ep), "S") && strings.Contains(strings.ToUpper(ep), "E") {
				var s, e int
				if _, err := fmt.Sscanf(strings.ToUpper(ep), "S%02dE%02d", &s, &e); err == nil {
					title = fmt.Sprintf("S%02dE%02d — %s", s, e, info.Title)
				} else {
					title = fmt.Sprintf("Ep %s: %s", ep, info.Title)
				}
			} else {
				title = fmt.Sprintf("Ep %s: %s", ep, info.Title)
			}
			var tags []string
			if info.Filler {
				tags = append(tags, "Filler")
			}
			if info.Recap {
				tags = append(tags, "Recap")
			}
			if info.Aired != "" {
				if info.Aired > today {
					tags = append(tags, "Unreleased (Airs "+info.Aired+")")
				} else {
					tags = append(tags, "Aired: "+info.Aired)
				}
			}
			desc = strings.Join(tags, " | ")
		} else {
			if strings.HasPrefix(strings.ToUpper(ep), "S") && strings.Contains(strings.ToUpper(ep), "E") {
				var s, e int
				if _, err := fmt.Sscanf(strings.ToUpper(ep), "S%02dE%02d", &s, &e); err == nil {
					title = fmt.Sprintf("Season %d, Episode %d", s, e)
				} else {
					title = fmt.Sprintf("Episode %s", ep)
				}
			} else {
				title = fmt.Sprintf("Episode %s", ep)
			}
		}

		subAvail := m.selectedShow.HasSub(ep)
		dubAvail := m.selectedShow.HasDub(ep)

		if strings.EqualFold(m.selectedShow.Provider, "vidsrc") || strings.EqualFold(m.selectedShow.Type, "MOVIE") || strings.EqualFold(m.selectedShow.Type, "TV") {
			subAvail = false
			dubAvail = false
		}

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

	var badge string
	if !strings.EqualFold(m.selectedShow.Provider, "vidsrc") && !strings.EqualFold(m.selectedShow.Type, "MOVIE") && !strings.EqualFold(m.selectedShow.Type, "TV") {
		if subCount > 0 && dubCount > 0 {
			badge = " [SUB+DUB]"
		} else if subCount > 0 {
			badge = " [SUB only]"
		} else if dubCount > 0 {
			badge = " [DUB only]"
		}
	}

	if strings.EqualFold(m.selectedShow.Type, "MOVIE") || strings.HasPrefix(m.selectedShow.ID, "vidsrc:movie") {
		m.episodeList.Title = "Select Stream"
	} else {
		m.episodeList.Title = fmt.Sprintf("Select Episode%s", badge)
	}
	m.episodeList.SetItems(items)
}
