package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type VidSrcProvider struct{}

func (p *VidSrcProvider) Name() string {
	return "vidsrc"
}

func (p *VidSrcProvider) Search(query, mode string) ([]AnimeShow, error) {
	tmdbSearchURL := fmt.Sprintf("https://api.themoviedb.org/3/search/multi?api_key=%s&query=%s", getTMDBApiKey(), url.QueryEscape(query))
	body, err := doHTTPReqWithRetry("GET", tmdbSearchURL, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("vidsrc search request failed: %w", err)
	}

	var searchResp struct {
		Results []struct {
			ID           int    `json:"id"`
			MediaType    string `json:"media_type"`
			Title        string `json:"title"`
			Name         string `json:"name"`
			OriginalName string `json:"original_name"`
			Overview     string `json:"overview"`
			ReleaseDate  string `json:"release_date"`
			FirstAirDate string `json:"first_air_date"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("vidsrc search parse failed: %w", err)
	}

	var shows []AnimeShow
	for _, item := range searchResp.Results {
		name := item.Title
		if name == "" {
			name = item.Name
		}
		if name == "" {
			name = item.OriginalName
		}
		if name == "" {
			continue
		}

		mediaType := item.MediaType
		if mediaType != "movie" && mediaType != "tv" {
			continue
		}

		year := 0
		dateStr := item.ReleaseDate
		if dateStr == "" {
			dateStr = item.FirstAirDate
		}
		if len(dateStr) >= 4 {
			fmt.Sscanf(dateStr[:4], "%d", &year)
		}

		showID := fmt.Sprintf("vidsrc:%s:%d", mediaType, item.ID)
		show := AnimeShow{
			ID:          showID,
			Name:        name,
			Description: item.Overview,
			Provider:    "vidsrc",
			Type:        strings.ToUpper(mediaType),
		}
		show.Season.Year = FlexInt(year)
		shows = append(shows, show)
	}

	return shows, nil
}

func (p *VidSrcProvider) FetchEpisodes(showID, mode string) (AnimeShow, []string, error) {
	cleanID := strings.TrimPrefix(showID, "vidsrc:")
	cleanID = strings.TrimPrefix(cleanID, "flikhub:")
	parts := strings.Split(cleanID, ":")
	var mediaType, tmdbID string
	if len(parts) >= 2 {
		mediaType = parts[0]
		tmdbID = parts[1]
	} else {
		mediaType = "movie"
		tmdbID = cleanID
	}

	if mediaType == "movie" {
		tmdbMovieURL := fmt.Sprintf("https://api.themoviedb.org/3/movie/%s?api_key=%s", tmdbID, getTMDBApiKey())
		body, err := doHTTPReqWithRetry("GET", tmdbMovieURL, nil, nil)
		if err == nil {
			var movieDetail struct {
				Title       string  `json:"title"`
				Overview    string  `json:"overview"`
				PosterPath  string  `json:"poster_path"`
				ReleaseDate string  `json:"release_date"`
				VoteAverage float64 `json:"vote_average"`
				Runtime     int     `json:"runtime"`
			}
			if json.Unmarshal(body, &movieDetail) == nil && movieDetail.Title != "" {
				thumb := ""
				if movieDetail.PosterPath != "" {
					thumb = "https://image.tmdb.org/t/p/w500" + movieDetail.PosterPath
				}
				year := 0
				if len(movieDetail.ReleaseDate) >= 4 {
					fmt.Sscanf(movieDetail.ReleaseDate[:4], "%d", &year)
				}
				durationStr := "Feature Film"
				if movieDetail.Runtime > 0 {
					durationStr = fmt.Sprintf("%d min", movieDetail.Runtime)
				}
				return AnimeShow{
					ID:          showID,
					Name:        movieDetail.Title,
					EnglishName: movieDetail.Title,
					Description: movieDetail.Overview,
					Thumbnail:   thumb,
					Type:        "MOVIE",
					Duration:    durationStr,
					Score:       movieDetail.VoteAverage,
					Season: struct {
						Quarter string  `json:"quarter"`
						Year    FlexInt `json:"year"`
					}{Year: FlexInt(year)},
					Provider: "vidsrc",
				}, []string{"1"}, nil
			}
		}
		return AnimeShow{
			ID:          showID,
			Name:        "Movie",
			Type:        "MOVIE",
			Duration:    "Feature Film",
			Provider:    "vidsrc",
		}, []string{"1"}, nil
	}

	tmdbDetailURL := fmt.Sprintf("https://api.themoviedb.org/3/tv/%s?api_key=%s", tmdbID, getTMDBApiKey())
	body, err := doHTTPReqWithRetry("GET", tmdbDetailURL, nil, nil)
	if err != nil {
		return AnimeShow{}, nil, fmt.Errorf("vidsrc tv details failed: %w", err)
	}

	var tvDetail struct {
		Name             string  `json:"name"`
		Overview         string  `json:"overview"`
		PosterPath       string  `json:"poster_path"`
		FirstAirDate     string  `json:"first_air_date"`
		VoteAverage      float64 `json:"vote_average"`
		NumberOfEpisodes int     `json:"number_of_episodes"`
		NumberOfSeasons  int     `json:"number_of_seasons"`
		Seasons          []struct {
			SeasonNumber int    `json:"season_number"`
			Name         string `json:"name"`
			EpisodeCount int    `json:"episode_count"`
			AirDate      string `json:"air_date"`
			PosterPath   string `json:"poster_path"`
		} `json:"seasons"`
	}

	if err := json.Unmarshal(body, &tvDetail); err != nil {
		return AnimeShow{}, nil, fmt.Errorf("vidsrc tv details parse failed: %w", err)
	}

	today := time.Now().Format("2006-01-02")
	var epList []string
	var seasonSummaries []SeasonSummary
	for _, season := range tvDetail.Seasons {
		if season.SeasonNumber <= 0 {
			continue
		}
		seasonName := season.Name
		if seasonName == "" {
			seasonName = fmt.Sprintf("Season %d", season.SeasonNumber)
		}
		isUnreleased := season.AirDate != "" && season.AirDate > today
		seasonSummaries = append(seasonSummaries, SeasonSummary{
			SeasonNumber: season.SeasonNumber,
			Name:         seasonName,
			EpisodeCount: season.EpisodeCount,
			AirDate:      season.AirDate,
			PosterPath:   season.PosterPath,
			Unreleased:   isUnreleased,
		})
		for e := 1; e <= season.EpisodeCount; e++ {
			epList = append(epList, fmt.Sprintf("S%02dE%02d", season.SeasonNumber, e))
		}
	}

	if len(epList) == 0 {
		epList = []string{"S01E01"}
	}

	thumb := ""
	if tvDetail.PosterPath != "" {
		thumb = "https://image.tmdb.org/t/p/w500" + tvDetail.PosterPath
	}
	year := 0
	if len(tvDetail.FirstAirDate) >= 4 {
		fmt.Sscanf(tvDetail.FirstAirDate[:4], "%d", &year)
	}

	durationStr := ""
	if tvDetail.NumberOfSeasons > 0 && tvDetail.NumberOfEpisodes > 0 {
		durationStr = fmt.Sprintf("%d Seasons (%d eps)", tvDetail.NumberOfSeasons, tvDetail.NumberOfEpisodes)
	} else if tvDetail.NumberOfEpisodes > 0 {
		durationStr = fmt.Sprintf("%d Episodes", tvDetail.NumberOfEpisodes)
	}

	return AnimeShow{
		ID:          showID,
		Name:        tvDetail.Name,
		EnglishName: tvDetail.Name,
		Description: tvDetail.Overview,
		Thumbnail:   thumb,
		Type:        "TV",
		Duration:    durationStr,
		Score:       tvDetail.VoteAverage,
		Season: struct {
			Quarter string  `json:"quarter"`
			Year    FlexInt `json:"year"`
		}{Year: FlexInt(year)},
		Seasons:  seasonSummaries,
		Provider: "vidsrc",
	}, epList, nil
}

func (p *VidSrcProvider) ResolveStreams(showID, mode, episodeNo, quality string) ([]ResolvedStream, error) {
	cleanID := strings.TrimPrefix(showID, "vidsrc:")
	cleanID = strings.TrimPrefix(cleanID, "flikhub:")
	parts := strings.Split(cleanID, ":")
	var mediaType, tmdbID string
	if len(parts) >= 2 {
		mediaType = parts[0]
		tmdbID = parts[1]
	} else {
		mediaType = "movie"
		tmdbID = cleanID
	}

	// Determine season from episodeNo (encoded as "S01E01" or a flat counter).
	// For the vaplayer API we always need season + episode numbers.
	season := "1"
	episode := episodeNo
	if strings.HasPrefix(strings.ToUpper(episodeNo), "S") {
		// "S01E03" style
		var s, e int
		if _, err := fmt.Sscanf(strings.ToUpper(episodeNo), "S%dE%d", &s, &e); err == nil {
			season = fmt.Sprintf("%d", s)
			episode = fmt.Sprintf("%d", e)
		}
	}

	streamURLs, subs, err := resolveVaplayerStream(tmdbID, mediaType, season, episode)
	if err != nil {
		return nil, fmt.Errorf("vidsrc stream resolution failed: %w", err)
	}
	if len(streamURLs) == 0 {
		return nil, fmt.Errorf("vidsrc: no stream URLs returned for %s", showID)
	}

	var streams []ResolvedStream
	for i, u := range streamURLs {
		quality := "HD"
		if i > 0 {
			quality = fmt.Sprintf("HD-Alt%d", i)
		}
		streams = append(streams, ResolvedStream{
			Provider:   "vidsrc",
			SourceName: fmt.Sprintf("VidSrc-%s", quality),
			Quality:    quality,
			URL:        u,
			Referer:    "https://nextgencloudfabric.com/",
			Subtitles:  subs,
		})
	}
	return streams, nil
}

// resolveVaplayerStream calls the streamdata.vaplayer.ru API which returns
// direct HLS m3u8 URLs and subtitle SRTs for movies and TV episodes by TMDB ID.
func resolveVaplayerStream(tmdbID, mediaType, season, episode string) ([]string, []SubtitleTrack, error) {
	var apiURL string
	if mediaType == "movie" {
		apiURL = fmt.Sprintf("https://streamdata.vaplayer.ru/api.php?tmdb=%s&type=movie", tmdbID)
	} else {
		apiURL = fmt.Sprintf("https://streamdata.vaplayer.ru/api.php?tmdb=%s&type=tv&season=%s&episode=%s", tmdbID, season, episode)
	}

	body, err := doHTTPReqWithRetry("GET", apiURL, nil, map[string]string{
		"User-Agent": UserAgent,
		"Referer":    "https://nextgencloudfabric.com/",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("vaplayer API request failed: %w", err)
	}

	var resp struct {
		StatusCode string `json:"status_code"`
		Data       struct {
			StreamURLs []string `json:"stream_urls"`
		} `json:"data"`
		DefaultSubs []struct {
			Lang string `json:"lang"`
			Code string `json:"code"`
			URL  string `json:"url"`
		} `json:"default_subs"`
		Subtitles []struct {
			Lang  string `json:"lang"`
			Code  string `json:"code"`
			URL   string `json:"url"`
			File  string `json:"file"`
			Label string `json:"label"`
		} `json:"subtitles"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, fmt.Errorf("vaplayer API parse failed: %w", err)
	}
	if resp.StatusCode != "200" && resp.StatusCode != "" {
		return nil, nil, fmt.Errorf("vaplayer API returned status %s", resp.StatusCode)
	}
	if len(resp.Data.StreamURLs) == 0 {
		return nil, nil, fmt.Errorf("vaplayer API returned no stream URLs")
	}

	var subs []SubtitleTrack
	for _, s := range resp.DefaultSubs {
		subURL := s.URL
		if subURL != "" {
			label := s.Lang
			if label == "" {
				label = s.Code
			}
			subs = append(subs, SubtitleTrack{
				Label: label,
				URL:   subURL,
			})
		}
	}
	for _, s := range resp.Subtitles {
		subURL := s.URL
		if subURL == "" {
			subURL = s.File
		}
		if subURL != "" {
			label := s.Label
			if label == "" {
				label = s.Lang
			}
			if label == "" {
				label = s.Code
			}
			subs = append(subs, SubtitleTrack{
				Label: label,
				URL:   subURL,
			})
		}
	}

	return resp.Data.StreamURLs, subs, nil
}
