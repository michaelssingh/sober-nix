package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func getTMDBApiKey() string {
	if k := os.Getenv("TMDB_API_KEY"); k != "" {
		return k
	}
	b, _ := base64.StdEncoding.DecodeString("NGU0NGQ5MDI5YjEyNzBhNzU3Y2RkYzc2NmExYmNiNjM=")
	return string(b)
}



var (
	linkPriorities = []string{"1080p", "720p", "480p", "360p", "best"}
)

type MediaType string

const (
	MediaTypeAnime  MediaType = "anime"
	MediaTypeMovie  MediaType = "movie"
	MediaTypeTV     MediaType = "tv"
	MediaTypeManga  MediaType = "manga"
	MediaTypeSports MediaType = "sports"
)

type MediaItem struct {
	ID                      string              `json:"_id"`
	Provider                string              `json:"provider"`
	Name                    string              `json:"name"`
	AvailableEpisodes       any                 `json:"availableEpisodes"`
	AvailableEpisodesDetail map[string][]string `json:"availableEpisodesDetail"`
	EnglishName             string              `json:"englishName"`
	NativeName              string              `json:"nativeName"`
	Thumbnail               string              `json:"thumbnail"`
	Description             string              `json:"description"`
	MALID                   string              `json:"malId"`
	AniListID               string              `json:"aniListId"`
	Type                    string              `json:"type"`
	Score                   float64             `json:"score"`
	Season                  struct {
		Quarter string  `json:"quarter"`
		Year    FlexInt `json:"year"`
	} `json:"season"`
	Duration string `json:"duration"`

	// Unified Media extension fields
	MediaType MediaType       `json:"media_type"`
	TMDBID    string          `json:"tmdb_id,omitempty"`
	Genres    []string        `json:"genres,omitempty"`
	Rating    string          `json:"rating,omitempty"`
	Seasons   []SeasonSummary `json:"seasons,omitempty"`
}

type SeasonSummary struct {
	SeasonNumber int    `json:"season_number"`
	Name         string `json:"name"`
	EpisodeCount int    `json:"episode_count"`
	AirDate      string `json:"air_date"`
	PosterPath   string `json:"poster_path"`
	Unreleased   bool   `json:"unreleased,omitempty"`
}

type AnimeShow = MediaItem

func (s AnimeShow) EpCount() int {
	if s.AvailableEpisodesDetail != nil {
		if eps, ok := s.AvailableEpisodesDetail["sub"]; ok {
			return len(eps)
		}
	}
	if s.AvailableEpisodes == nil {
		return 0
	}
	if episodes, ok := s.AvailableEpisodes.(map[string]any); ok {
		if subEpisodes, ok := episodes["sub"].(float64); ok {
			return int(subEpisodes)
		}
	}
	return 0
}

func (s AnimeShow) SubCount() int {
	if s.AvailableEpisodesDetail != nil {
		if eps, ok := s.AvailableEpisodesDetail["sub"]; ok {
			return len(eps)
		}
	}
	if s.AvailableEpisodes == nil {
		return 0
	}
	if episodes, ok := s.AvailableEpisodes.(map[string]any); ok {
		if subEpisodes, ok := episodes["sub"].(float64); ok {
			return int(subEpisodes)
		}
	}
	return 0
}

func (s AnimeShow) DubCount() int {
	if s.AvailableEpisodesDetail != nil {
		if eps, ok := s.AvailableEpisodesDetail["dub"]; ok {
			return len(eps)
		}
	}
	if s.AvailableEpisodes == nil {
		return 0
	}
	if episodes, ok := s.AvailableEpisodes.(map[string]any); ok {
		if dubEpisodes, ok := episodes["dub"].(float64); ok {
			return int(dubEpisodes)
		}
	}
	return 0
}

func (s AnimeShow) HasSub(ep string) bool {
	if s.AvailableEpisodesDetail != nil {
		if eps, ok := s.AvailableEpisodesDetail["sub"]; ok {
			for _, e := range eps {
				if e == ep {
					return true
				}
			}
			return false
		}
	}
	epVal := parseEpisodeNumber(ep)
	subCount := s.SubCount()
	return subCount > 0 && epVal <= float64(subCount)
}

func (s AnimeShow) HasDub(ep string) bool {
	if s.AvailableEpisodesDetail != nil {
		if eps, ok := s.AvailableEpisodesDetail["dub"]; ok {
			for _, e := range eps {
				if e == ep {
					return true
				}
			}
			return false
		}
	}
	epVal := parseEpisodeNumber(ep)
	dubCount := s.DubCount()
	return dubCount > 0 && epVal <= float64(dubCount)
}

type FlexInt int

func (fi *FlexInt) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	var i int
	if err := json.Unmarshal(b, &i); err == nil {
		*fi = FlexInt(i)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		var val int
		if _, err := fmt.Sscanf(s, "%d", &val); err == nil {
			*fi = FlexInt(val)
		}
		return nil
	}
	return nil
}

type SourceInfo struct {
	SourceName string
	SourceURL  string
}

type LinkEntry struct {
	Link          string `json:"link"`
	ResolutionStr string `json:"resolutionStr"`
	HLS           bool   `json:"hls"`
	Mp4           bool   `json:"mp4"`
}

type ClockResponse struct {
	Links []LinkEntry `json:"links"`
}

var (
	streamCache        = make(map[string]string)
	streamCacheMu      sync.RWMutex
	activePrefetches   = make(map[string]bool)
	activePrefetchesMu sync.Mutex
)

var providers = []Provider{
	&AniDBProvider{},
	&VidSrcProvider{},
}

func getProvider(name string) Provider {
	for _, p := range providers {
		if strings.EqualFold(p.Name(), name) {
			return p
		}
	}
	return &AniDBProvider{}
}

func rankTitle(name, query string) int {
	if name == "" {
		return 3
	}
	nameLower := strings.ToLower(name)
	queryLower := strings.ToLower(query)
	if nameLower == queryLower {
		return 0
	}
	if strings.HasPrefix(nameLower, queryLower) {
		return 1
	}
	if strings.Contains(nameLower, queryLower) {
		return 2
	}
	return 3
}

func rankSearchRelevance(show AnimeShow, query string) int {
	best := rankTitle(show.Name, query)
	if r := rankTitle(show.EnglishName, query); r < best {
		best = r
	}
	if r := rankTitle(show.NativeName, query); r < best {
		best = r
	}
	return best
}

func searchAnime(query, mode string) ([]AnimeShow, error) {
	var results []AnimeShow
	var mu sync.Mutex
	var wg sync.WaitGroup

	cfg := loadConfig()
	for _, p := range providers {
		if !cfg.IsProviderEnabled(p.Name()) {
			continue
		}
		wg.Add(1)
		go func(prov Provider) {
			defer wg.Done()
			shows, err := prov.Search(query, mode)
			if err == nil {
				mu.Lock()
				for i := range shows {
					shows[i].Provider = prov.Name()
					if !strings.HasPrefix(shows[i].ID, prov.Name()+":") {
						shows[i].ID = prov.Name() + ":" + shows[i].ID
					}
					_ = saveShowCache(shows[i].ID, shows[i], nil)
				}
				results = append(results, shows...)
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()

	var filtered []AnimeShow
	queryWords := strings.Fields(strings.ToLower(query))
	for _, show := range results {
		rank := rankSearchRelevance(show, query)
		if rank < 3 {
			filtered = append(filtered, show)
		} else {
			// Check if at least one query word matches
			titleLower := strings.ToLower(show.Name + " " + show.EnglishName + " " + show.NativeName)
			matched := false
			for _, qw := range queryWords {
				if len(qw) > 2 && strings.Contains(titleLower, qw) {
					matched = true
					break
				}
			}
			if matched {
				filtered = append(filtered, show)
			}
		}
	}
	results = filtered

	sort.SliceStable(results, func(i, j int) bool {
		rankI := rankSearchRelevance(results[i], query)
		rankJ := rankSearchRelevance(results[j], query)
		if rankI != rankJ {
			return rankI < rankJ
		}
		// Provider priority: anime-native providers first, then movie/TV
		provOrder := map[string]int{"allanime": 0, "anidb": 1, "flikhub": 2, "gogoanime": 3, "vidsrc": 4}
		pi, piOk := provOrder[strings.ToLower(results[i].Provider)]
		pj, pjOk := provOrder[strings.ToLower(results[j].Provider)]
		if !piOk {
			pi = 99
		}
		if !pjOk {
			pj = 99
		}
		if pi != pj {
			return pi < pj
		}
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	// Deduplicate: keep one entry per (normalised-title, type) pair.
	// After sorting the preferred provider is already first, so we keep the
	// first occurrence and discard later duplicates with the same title+type.
	seen := make(map[string]struct{})
	var deduped []AnimeShow
	for _, show := range results {
		enrichShowMetadata(&show)
		_ = saveShowCache(show.ID, show, nil)
		key := strings.ToLower(strings.TrimSpace(show.Name)) + "|" + strings.ToLower(show.Type)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, show)
	}
	results = deduped

	return results, nil
}

func fetchEpisodeList(showID, mode string) (AnimeShow, []string, error) {
	cfg := loadConfig()
	provider := ""
	if cached, _, found := loadShowCache(showID); found && cached.Provider != "" {
		provider = cached.Provider
	}
	if provider == "" {
		// Infer provider by showID format or prefix
		if strings.HasPrefix(showID, "anidb:") {
			provider = "anidb"
		} else if strings.HasPrefix(showID, "flikhub:") {
			provider = "flikhub"
		} else if strings.HasPrefix(showID, "allanime:") {
			provider = "allanime"
		} else if strings.HasPrefix(showID, "vidsrc:") {
			provider = "vidsrc"
		} else if strings.HasSuffix(showID, "-online") || strings.Contains(showID, "-1") {
			provider = "gogoanime"
		} else if strings.Contains(showID, "-") {
			provider = "flikhub"
		} else {
			provider = "allanime"
		}
	}
	if !cfg.IsProviderEnabled(provider) {
		return AnimeShow{}, nil, fmt.Errorf("provider %q is disabled", provider)
	}
	p := getProvider(provider)
	show, eps, err := p.FetchEpisodes(showID, mode)
	if err == nil {
		if show.MALID == "" || show.Thumbnail == "" {
			enrichShowMetadata(&show)
		}
		_ = saveShowCache(showID, show, eps)
	}
	return show, eps, err
}

func enrichShowMetadata(show *AnimeShow) {
	cleanTitle := show.Name
	if idx := strings.Index(cleanTitle, "("); idx > 0 {
		cleanTitle = strings.TrimSpace(cleanTitle[:idx])
	}
	if cleanTitle == "" || cleanTitle == "AniDB Show" {
		return
	}

	query := `query ($search: String) {
		Media(search: $search, type: ANIME) {
			id
			idMal
			format
			episodes
			duration
			title {
				romaji
				english
				native
			}
			coverImage {
				extraLarge
				large
			}
			description
			seasonYear
			score: averageScore
			genres
		}
	}`
	payload := map[string]any{
		"query": query,
		"variables": map[string]any{
			"search": cleanTitle,
		},
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   UserAgent,
	}

	respBody, err := doHTTPReqWithRetry("POST", "https://graphql.anilist.co", jsonBody, headers)
	if err != nil {
		return
	}

	var alResp struct {
		Data struct {
			Media struct {
				ID          int      `json:"id"`
				IDMal       int      `json:"idMal"`
				Format      string   `json:"format"`
				Episodes    int      `json:"episodes"`
				Duration    int      `json:"duration"`
				Description string   `json:"description"`
				SeasonYear  int      `json:"seasonYear"`
				Score       float64  `json:"score"`
				Genres      []string `json:"genres"`
				Title       struct {
					Romaji  string `json:"romaji"`
					English string `json:"english"`
					Native  string `json:"native"`
				} `json:"title"`
				CoverImage struct {
					ExtraLarge string `json:"extraLarge"`
					Large      string `json:"large"`
				} `json:"coverImage"`
			} `json:"Media"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &alResp); err == nil && alResp.Data.Media.ID != 0 {
		m := alResp.Data.Media
		if show.MALID == "" && m.IDMal != 0 {
			show.MALID = strconv.Itoa(m.IDMal)
		}
		if show.AniListID == "" && m.ID != 0 {
			show.AniListID = strconv.Itoa(m.ID)
		}
		if show.Thumbnail == "" {
			if m.CoverImage.ExtraLarge != "" {
				show.Thumbnail = m.CoverImage.ExtraLarge
			} else {
				show.Thumbnail = m.CoverImage.Large
			}
		}
		if show.Description == "" || strings.HasPrefix(show.Description, "anidb:") || (!strings.Contains(show.Description, " ") && len(show.Description) < 30) {
			show.Description = m.Description
		}
		if show.EnglishName == "" && m.Title.English != "" {
			show.EnglishName = m.Title.English
		}
		if show.NativeName == "" && m.Title.Native != "" {
			show.NativeName = m.Title.Native
		}
		if show.Score == 0 && m.Score != 0 {
			show.Score = m.Score / 10.0
		}
		if show.Season.Year == 0 && m.SeasonYear != 0 {
			show.Season.Year = FlexInt(m.SeasonYear)
		}
		if show.Type == "" && m.Format != "" {
			show.Type = m.Format
		}
		if len(show.Genres) == 0 && len(m.Genres) > 0 {
			show.Genres = m.Genres
		}
		if show.Duration == "" {
			if m.Episodes > 0 && m.Duration > 0 {
				show.Duration = fmt.Sprintf("%d eps × %d min", m.Episodes, m.Duration)
			} else if m.Duration > 0 {
				show.Duration = fmt.Sprintf("%d min", m.Duration)
			} else if m.Episodes > 0 {
				show.Duration = fmt.Sprintf("%d Episodes", m.Episodes)
			}
		}
	}
}

func resolveMALIDFromTitle(title string) string {
	cleanTitle := title
	if idx := strings.Index(cleanTitle, "("); idx > 0 {
		cleanTitle = strings.TrimSpace(cleanTitle[:idx])
	}
	if cleanTitle == "" || cleanTitle == "AniDB Show" {
		return ""
	}

	query := `query ($search: String) {
		Media(search: $search, type: ANIME) {
			idMal
		}
	}`
	payload := map[string]any{
		"query": query,
		"variables": map[string]any{
			"search": cleanTitle,
		},
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return ""
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   UserAgent,
	}

	respBody, err := doHTTPReqWithRetry("POST", "https://graphql.anilist.co", jsonBody, headers)
	if err != nil {
		return ""
	}

	var res struct {
		Data struct {
			Media struct {
				IDMal int `json:"idMal"`
			} `json:"Media"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &res); err == nil && res.Data.Media.IDMal > 0 {
		return fmt.Sprintf("%d", res.Data.Media.IDMal)
	}
	return ""
}

func prefetchEpisodeStream(showID, mode, epNo, quality string) {
	if showID == "" || epNo == "" {
		return
	}
	cacheKey := fmt.Sprintf("%s-%s-%s-%s", showID, mode, epNo, quality)

	streamCacheMu.RLock()
	_, cached := streamCache[cacheKey]
	streamCacheMu.RUnlock()
	if cached {
		return
	}

	activePrefetchesMu.Lock()
	if activePrefetches[cacheKey] {
		activePrefetchesMu.Unlock()
		return
	}
	activePrefetches[cacheKey] = true
	activePrefetchesMu.Unlock()

	go func() {
		defer func() {
			activePrefetchesMu.Lock()
			delete(activePrefetches, cacheKey)
			activePrefetchesMu.Unlock()
		}()

		debugLog("prefetchEpisodeStream: starting background prefetch for %s", cacheKey)
		if mode == "dual" {
			_, _ = resolveStreamURL(showID, "sub", epNo, quality)
			_, _ = resolveStreamURL(showID, "dub", epNo, quality)
		} else {
			_, _ = resolveStreamURL(showID, mode, epNo, quality)
		}
	}()
}

func resolveStreamURL(showID, mode, episodeNo, quality string) (string, error) {
	cacheKey := fmt.Sprintf("%s-%s-%s-%s", showID, mode, episodeNo, quality)
	streamCacheMu.RLock()
	if val, ok := streamCache[cacheKey]; ok {
		streamCacheMu.RUnlock()
		debugLog("resolveStreamURL: cache hit for %s", cacheKey)
		return val, nil
	}
	streamCacheMu.RUnlock()

	streams, err := fetchAllResolvedStreams(showID, mode, episodeNo, "")
	if err != nil {
		return "", err
	}

	best := selectBestLinkFromResolved(streams, quality)
	if best != "" {
		streamCacheMu.Lock()
		streamCache[cacheKey] = best
		streamCacheMu.Unlock()
		return best, nil
	}

	return "", fmt.Errorf("no stream links resolved for episode %s (%s)", episodeNo, mode)
}

func selectBestLinkFromResolved(streams []ResolvedStream, requested string) string {
	if len(streams) == 0 {
		return ""
	}
	priorities := []string{"1080p", "720p", "480p", "360p", "hls", "best"}
	if requested == "worst" {
		priorities = []string{"360p", "480p", "720p", "1080p", "hls", "best"}
	} else if requested != "best" && requested != "" {
		for _, s := range streams {
			if strings.EqualFold(s.Quality, requested) {
				return s.URL
			}
		}
	}
	for _, p := range priorities {
		for _, s := range streams {
			if strings.EqualFold(s.Quality, p) {
				return s.URL
			}
		}
	}
	return streams[0].URL
}





type SubtitleTrack struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type ResolvedStream struct {
	Provider   string
	SourceName string
	Quality    string
	URL        string
	Referer    string
	Subtitles  []SubtitleTrack
}



func fetchAllResolvedStreams(showID, mode, episodeNo, providerName string) ([]ResolvedStream, error) {
	var results []ResolvedStream
	var mu sync.Mutex
	var wg sync.WaitGroup

	cfg := loadConfig()
	if providerName == "" && strings.Contains(showID, ":") {
		providerName = strings.SplitN(showID, ":", 2)[0]
	}

	targetProviders := providers
	if providerName != "" {
		for _, p := range providers {
			if strings.EqualFold(p.Name(), providerName) {
				targetProviders = []Provider{p}
				break
			}
		}
	}

	cleanID := stripProviderPrefix(showID)

	for _, p := range targetProviders {
		if !cfg.IsProviderEnabled(p.Name()) {
			debugLog("fetchAllResolvedStreams: skipping disabled provider %s", p.Name())
			continue
		}
		wg.Add(1)
		go func(prov Provider) {
			defer wg.Done()
			streams, err := prov.ResolveStreams(cleanID, mode, episodeNo, "best")
			if err == nil && len(streams) > 0 {
				mu.Lock()
				for _, s := range streams {
					if s.Provider == "" {
						s.Provider = prov.Name()
					}
					results = append(results, s)
				}
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()

	if len(results) == 0 {
		return nil, fmt.Errorf("no streams resolved for episode %s (%s)", episodeNo, mode)
	}

	return results, nil
}

type AiringShow struct {
	ID           int    `json:"id"`
	IDMal        int    `json:"idMal"`
	TitleRomaji  string `json:"titleRomaji"`
	TitleEnglish string `json:"titleEnglish"`
	TitleNative  string `json:"titleNative"`
	Description  string `json:"description"`
	Episodes     int    `json:"episodes"`
	CoverImage   string `json:"coverImage"`
	NextEpisode  int    `json:"nextEpisode"`
	TimeUntil    int    `json:"timeUntil"`
}

func fetchAiringAnime() ([]AiringShow, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	query := `
	query {
	  Page(page: 1, perPage: 20) {
	    media(status: RELEASING, sort: POPULARITY_DESC, type: ANIME) {
	      id
	      idMal
	      title {
	        romaji
	        english
	        native
	      }
	      coverImage {
	        large
	      }
	      description
	      episodes
	      nextAiringEpisode {
	        episode
	        timeUntilAiring
	      }
	    }
	  }
	}`

	payload := map[string]interface{}{
		"query": query,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://graphql.anilist.co", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AniList airing request returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var aniResp struct {
		Data struct {
			Page struct {
				Media []struct {
					ID    int   `json:"id"`
					IDMal *int  `json:"idMal"`
					Title struct {
						Romaji  string `json:"romaji"`
						English string `json:"english"`
						Native  string `json:"native"`
					} `json:"title"`
					CoverImage struct {
						Large string `json:"large"`
					} `json:"coverImage"`
					Description       string `json:"description"`
					Episodes          *int   `json:"episodes"`
					NextAiringEpisode *struct {
						Episode          int `json:"episode"`
						TimeUntilAiring int `json:"timeUntilAiring"`
					} `json:"nextAiringEpisode"`
				} `json:"media"`
			} `json:"Page"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&aniResp); err != nil {
		return nil, err
	}

	var shows []AiringShow
	for _, m := range aniResp.Data.Page.Media {
		malID := 0
		if m.IDMal != nil {
			malID = *m.IDMal
		}
		eps := 0
		if m.Episodes != nil {
			eps = *m.Episodes
		}
		nextEp := 0
		timeUntil := 0
		if m.NextAiringEpisode != nil {
			nextEp = m.NextAiringEpisode.Episode
			timeUntil = m.NextAiringEpisode.TimeUntilAiring
		}

		shows = append(shows, AiringShow{
			ID:           m.ID,
			IDMal:        malID,
			TitleRomaji:  m.Title.Romaji,
			TitleEnglish: m.Title.English,
			TitleNative:  m.Title.Native,
			Description:  m.Description,
			Episodes:     eps,
			CoverImage:   m.CoverImage.Large,
			NextEpisode:  nextEp,
			TimeUntil:    timeUntil,
		})
	}
	return shows, nil
}

func checkStreamMpvDryRun(urlVal string, headers map[string]string) (bool, error) {
	useYtdl := false
	lowerURL := strings.ToLower(urlVal)
	if strings.Contains(lowerURL, "ok.ru") ||
		strings.Contains(lowerURL, "filemoon") ||
		strings.Contains(lowerURL, "bysekoze") ||
		strings.Contains(lowerURL, "streamwish") ||
		strings.Contains(lowerURL, "listeamed") ||
		strings.Contains(lowerURL, "vidguard") {
		useYtdl = true
	}

	args := []string{
		"--vo=null",
		"--ao=null",
		"--length=5",
		"--msg-level=all=status",
		"--network-timeout=5",
	}
	if !useYtdl {
		args = append(args, "--ytdl=no")
	}
	var headerFields []string
	for k, v := range headers {
		headerFields = append(headerFields, fmt.Sprintf("%s: %s", k, v))
	}
	if len(headerFields) > 0 {
		args = append(args, fmt.Sprintf("--http-header-fields=%s", strings.Join(headerFields, ",")))
	}
	args = append(args, urlVal)

	cmd := exec.Command("mpv", args...)
	err := cmd.Run()
	if err != nil {
		return false, err
	}
	return true, nil
}
