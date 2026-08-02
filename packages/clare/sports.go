package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ─── Data types ──────────────────────────────────────────────────────────────

// SportsEvent represents a live or upcoming sporting event.
type SportsEvent struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Category  string `json:"category"` // e.g. "american-football", "football", "basketball", "motorsport", "fight"
	Teams     string `json:"teams"`    // "<Home> vs <Away>"
	StartTime string `json:"start_time"`
	IsLive    bool   `json:"is_live"`
	Sources   []SportsStream
}

// SportsStream is a single playable stream for a SportsEvent.
type SportsStream struct {
	ID         string `json:"id"`
	Name       string `json:"name"`    // e.g. "Sky Sports Main Event"
	Language   string `json:"lang"`    // e.g. "en"
	URL        string `json:"url"`
	Referer    string `json:"referer"`
	Resolution string `json:"resolution"`
}

// ─── Streamed.su provider ────────────────────────────────────────────────────

const streamedBaseURL = "https://streamed.st"

// fetchStreamedLiveMatches returns currently live events from Streamed.st.
func fetchStreamedLiveMatches() ([]SportsEvent, error) {
	body, err := doHTTPReqWithRetry("GET", streamedBaseURL+"/api/matches/live", nil, map[string]string{
		"Referer":    "https://streamed.st/",
		"User-Agent": UserAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("streamed.st live request failed: %w", err)
	}

	var raw []struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Category string `json:"category"`
		Date     int64  `json:"date"` // unix millis
		Teams    struct {
			Home struct {
				Name string `json:"name"`
			} `json:"home"`
			Away struct {
				Name string `json:"name"`
			} `json:"away"`
		} `json:"teams"`
		Sources []struct {
			Source string `json:"source"`
			ID     string `json:"id"`
		} `json:"sources"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("streamed.st parse failed: %w", err)
	}

	var events []SportsEvent
	for _, r := range raw {
		teams := r.Teams.Home.Name + " vs " + r.Teams.Away.Name
		if r.Teams.Home.Name == "" && r.Teams.Away.Name == "" {
			teams = ""
		}
		t := time.UnixMilli(r.Date).UTC()
		ev := SportsEvent{
			ID:        "streamed:" + r.ID,
			Title:     r.Title,
			Category:  normalizeCategory(r.Category),
			Teams:     teams,
			StartTime: t.Format("15:04 UTC"),
			IsLive:    true,
		}
		// Fetch stream sources for each match
		for _, src := range r.Sources {
			streams, err := fetchStreamedMatchStreams(r.ID, src.Source, src.ID)
			if err != nil {
				debugLog("sports: streamed stream fetch error for %s: %v", r.ID, err)
				continue
			}
			ev.Sources = append(ev.Sources, streams...)
		}
		if len(ev.Sources) > 0 {
			events = append(events, ev)
		}
	}
	return events, nil
}

// fetchStreamedMatchStreams fetches the actual HLS stream URLs for one source of a match.
func fetchStreamedMatchStreams(matchID, source, streamID string) ([]SportsStream, error) {
	url := fmt.Sprintf("%s/api/stream/%s/%s", streamedBaseURL, source, streamID)
	body, err := doHTTPReqWithRetry("GET", url, nil, map[string]string{
		"Referer":    "https://streamed.st/",
		"User-Agent": UserAgent,
	})
	if err != nil {
		return nil, err
	}

	var raw []struct {
		ID       string `json:"id"`
		StreamNo int    `json:"streamNo"`
		Lang     string `json:"lang"`
		HLS      string `json:"hls"`
		EmbedURL string `json:"embedUrl"`
		Qualities []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"qualities"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var streams []SportsStream
	for _, s := range raw {
		hls := s.HLS
		if hls == "" && s.EmbedURL != "" {
			hls = s.EmbedURL
		}
		name := fmt.Sprintf("%s #%d", strings.ToUpper(source), s.StreamNo)
		lang := s.Lang
		if lang == "" {
			lang = "en"
		}
		if len(s.Qualities) > 0 {
			hls = s.Qualities[0].URL
		}
		if hls == "" {
			continue
		}
		streams = append(streams, SportsStream{
			ID:       fmt.Sprintf("%s-%s-%d", matchID, source, s.StreamNo),
			Name:     name,
			Language: lang,
			URL:      hls,
			Referer:  "https://streamed.st/",
		})
	}
	return streams, nil
}


// ─── DaddyLive provider ───────────────────────────────────────────────────────

const daddyLiveBase = "https://dlhd.st"

// DaddyLiveScheduleEntry is a single scheduled event in DaddyLive's JSON.
type DaddyLiveScheduleEntry struct {
	Time     string `json:"time"`
	Title    string `json:"title"`
	Channels []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"channels"`
}

// fetchDaddyLiveSchedule returns today's scheduled events from DaddyLive.
func fetchDaddyLiveSchedule() ([]SportsEvent, error) {
	body, err := doHTTPReqWithRetry("GET", daddyLiveBase+"/schedule/schedule-generated.json", nil, map[string]string{
		"Referer":    daddyLiveBase + "/",
		"User-Agent": UserAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("daddylive schedule fetch failed: %w", err)
	}

	// The schedule JSON is keyed by date string, e.g. {"Thursday 24th July":{"Football":[...],...}}
	var raw map[string]map[string][]DaddyLiveScheduleEntry
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("daddylive schedule parse failed: %w", err)
	}

	var events []SportsEvent
	for _, categories := range raw {
		for category, entries := range categories {
			cleanCategory := strings.ReplaceAll(category, "</span>", "")
			cleanCategory = strings.ReplaceAll(cleanCategory, "<span>", "")
			cleanCategory = strings.TrimSpace(cleanCategory)
			for _, entry := range entries {
				title := entry.Title
				if title == "" || title == "None" {
					title = cleanCategory + " Event"
				}
				ev := SportsEvent{
					ID:        fmt.Sprintf("daddy:%s:%s", cleanCategory, title),
					Title:     title,
					Category:  normalizeCategory(cleanCategory),
					StartTime: entry.Time,
					IsLive:    false,
				}
				for _, ch := range entry.Channels {
					ev.Sources = append(ev.Sources, SportsStream{
						ID:      "daddy:" + ch.ID,
						Name:    ch.Name,
						Referer: daddyLiveBase + "/",
					})
				}
				events = append(events, ev)
			}
		}
	}
	return events, nil
}

// resolveDaddyLiveStream unpacks a DaddyLive channel page and returns its HLS URL.
func resolveDaddyLiveStream(channelID string) (string, error) {
	embedURL := fmt.Sprintf("%s/stream/stream-%s.php", daddyLiveBase, channelID)
	body, err := doHTTPReqWithRetry("GET", embedURL, nil, map[string]string{
		"Referer":    daddyLiveBase + "/",
		"User-Agent": UserAgent,
	})
	if err != nil {
		embedURL = fmt.Sprintf("%s/embed/stream-%s.php", daddyLiveBase, channelID)
		body, err = doHTTPReqWithRetry("GET", embedURL, nil, map[string]string{
			"Referer":    daddyLiveBase + "/",
			"User-Agent": UserAgent,
		})
		if err != nil {
			return "", fmt.Errorf("daddylive embed fetch failed: %w", err)
		}
	}

	html := string(body)

	// Try to find an m3u8 URL directly in the page
	m3u8Re := regexp.MustCompile(`https?://[^\s"'<>]+\.m3u8[^\s"'<>]*`)
	if m := m3u8Re.FindString(html); m != "" {
		return m, nil
	}

	// Try to find window.atob base64 encoded stream source
	atobRe := regexp.MustCompile(`window\.atob\s*\(\s*['"]([A-Za-z0-9+/=]+)['"]\s*\)`)
	if match := atobRe.FindStringSubmatch(html); len(match) > 1 {
		if decoded, err := base64.StdEncoding.DecodeString(match[1]); err == nil {
			decStr := string(decoded)
			if m := m3u8Re.FindString(decStr); m != "" {
				return m, nil
			}
		}
	}

	// Try to find an iframe src pointing to an HLS player
	iframeRe := regexp.MustCompile(`<iframe[^>]+src=["']([^"']+)["']`)
	matches := iframeRe.FindAllStringSubmatch(html, -1)
	for _, match := range matches {
		if len(match) > 1 {
			playerURL := match[1]
			if strings.HasPrefix(playerURL, "//") {
				playerURL = "https:" + playerURL
			}
			playerBody, err := doHTTPReqWithRetry("GET", playerURL, nil, map[string]string{
				"Referer":    embedURL,
				"User-Agent": UserAgent,
			})
			if err != nil {
				continue
			}
			playerHTML := string(playerBody)
			if m := m3u8Re.FindString(playerHTML); m != "" {
				return m, nil
			}
			if atobMatch := atobRe.FindStringSubmatch(playerHTML); len(atobMatch) > 1 {
				if decoded, err := base64.StdEncoding.DecodeString(atobMatch[1]); err == nil {
					decStr := string(decoded)
					if m := m3u8Re.FindString(decStr); m != "" {
						return m, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no HLS stream found for channel %s", channelID)
}

// ─── Combined fetcher ─────────────────────────────────────────────────────────

// FetchAllSportsEvents merges live sports events from active providers (DaddyLive & streamed.st).
func FetchAllSportsEvents() ([]SportsEvent, error) {
	var events []SportsEvent

	// 1. Fetch DaddyLive schedule events (290+ channels with direct 200 OK HLS streams)
	daddyEvents, errDL := fetchDaddyLiveSchedule()
	if errDL == nil {
		events = append(events, daddyEvents...)
	} else {
		debugLog("sports: daddylive error: %v", errDL)
	}

	// 2. Fetch Streamed.st live matches
	streamedEvents, errST := fetchStreamedLiveMatches()
	if errST == nil {
		events = append(events, streamedEvents...)
	} else {
		debugLog("sports: streamed.st error: %v", errST)
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("failed to fetch live sports from any provider")
	}

	// Sort: by category, then by title
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Category != events[j].Category {
			return events[i].Category < events[j].Category
		}
		return events[i].Title < events[j].Title
	})

	return events, nil
}

// ResolveSportsEventStream resolves the stream URL for a SportsStream.
// For DaddyLive events, we unpack the inner iframe base64 stream.
func ResolveSportsEventStream(stream SportsStream) (SportsStream, error) {
	if strings.HasPrefix(stream.ID, "daddy:") || stream.URL == "" {
		id := strings.TrimPrefix(stream.ID, "daddy:")
		hls, err := resolveDaddyLiveStream(id)
		if err == nil && hls != "" {
			stream.URL = hls
			return stream, nil
		}
	}

	if stream.URL != "" {
		return stream, nil
	}
	return SportsStream{}, fmt.Errorf("could not resolve playable stream for %s", stream.Name)
}

// ─── Category helpers ─────────────────────────────────────────────────────────

var categoryEmoji = map[string]string{
	"football":          "⚽",
	"american-football": "🏈",
	"basketball":        "🏀",
	"baseball":          "⚾",
	"hockey":            "🏒",
	"motorsport":        "🏎️",
	"fight":             "🥊",
	"tennis":            "🎾",
	"rugby":             "🏉",
	"cricket":           "🏏",
	"golf":              "⛳",
	"other":             "🏟️",
}

var categoryNormMap = map[string]string{
	"soccer":           "football",
	"american football": "american-football",
	"nfl":              "american-football",
	"nba":              "basketball",
	"mlb":              "baseball",
	"nhl":              "hockey",
	"f1":               "motorsport",
	"formula 1":        "motorsport",
	"formula one":      "motorsport",
	"mma":              "fight",
	"ufc":              "fight",
	"boxing":           "fight",
}

func normalizeCategory(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if norm, ok := categoryNormMap[lower]; ok {
		return norm
	}
	// If already a known category key, return it
	if _, ok := categoryEmoji[lower]; ok {
		return lower
	}
	// Replace spaces with dashes
	lower = strings.ReplaceAll(lower, " ", "-")
	if _, ok := categoryEmoji[lower]; ok {
		return lower
	}
	return "other"
}

// CategoryEmoji returns the emoji prefix for a category.
func CategoryEmoji(category string) string {
	if emoji, ok := categoryEmoji[category]; ok {
		return emoji
	}
	return "🏟️"
}

// SportsCategoryLabel returns a display label for a category.
func SportsCategoryLabel(category string) string {
	labels := map[string]string{
		"football":          "Soccer",
		"american-football": "American Football",
		"basketball":        "Basketball",
		"baseball":          "Baseball",
		"hockey":            "Hockey",
		"motorsport":        "Motorsport / F1",
		"fight":             "Combat Sports",
		"tennis":            "Tennis",
		"rugby":             "Rugby",
		"cricket":           "Cricket",
		"golf":              "Golf",
		"other":             "Other Sports",
	}
	if l, ok := labels[category]; ok {
		return l
	}
	return strings.Title(category)
}
