package main

type Provider interface {
	Name() string
	Search(query string, mode string) ([]AnimeShow, error)
	FetchEpisodes(showID string, mode string) (AnimeShow, []string, error)
	ResolveStreams(showID, mode, episodeNo, quality string) ([]ResolvedStream, error)
}
