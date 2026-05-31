package tyto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:150.0) Gecko/20100101 Firefox/150.0"
	Referer   = "https://youtu-chan.com"
	ApiBase   = "https://api.allanime.day/api"
)

func Decrypt(data string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil || len(decoded) < 29 {
		return "", fmt.Errorf("invalid payload")
	}

	key := sha256.Sum256([]byte("Xot36i3lK3:v1"))
	block, _ := aes.NewCipher(key[:])

	ctrBytes := make([]byte, 16)
	copy(ctrBytes, decoded[1:13])
	ctrBytes[15] = 2

	ciphertext := decoded[13 : len(decoded)-16]
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCTR(block, ctrBytes).XORKeyStream(plaintext, ciphertext)

	return string(plaintext), nil
}

func queryGraphQL(query string, variables map[string]any, response any) error {
	body, _ := json.Marshal(map[string]any{"query": query, "variables": variables})
	req, _ := http.NewRequest("POST", ApiBase, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Referer", Referer)
	req.Header.Set("Origin", Referer)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if b, ok := response.(*[]byte); ok {
		*b, err = io.ReadAll(resp.Body)
		return err
	}
	return json.NewDecoder(resp.Body).Decode(response)
}

type Anime struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
}

func SearchAnime(query, mode string) ([]Anime, error) {
	var result struct {
		Data struct {
			Shows struct {
				Edges []Anime `json:"edges"`
			} `json:"shows"`
		} `json:"data"`
	}

	err := queryGraphQL(
		`query( $search: SearchInput $limit: Int $page: Int $translationType: VaildTranslationTypeEnumType $countryOrigin: VaildCountryOriginEnumType ) { shows( search: $search limit: $limit page: $page translationType: $translationType countryOrigin: $countryOrigin ) { edges { _id name availableEpisodes __typename } }}`,
		map[string]any{"search": map[string]any{"allowAdult": false, "allowUnknown": false, "query": query}, "limit": 40, "page": 1, "translationType": mode, "countryOrigin": "ALL"},
		&result,
	)
	return result.Data.Shows.Edges, err
}

func GetEpisodes(showID, mode string) ([]string, error) {
	var result struct {
		Data struct {
			Show struct {
				Available map[string][]string `json:"availableEpisodesDetail"`
			} `json:"show"`
		} `json:"data"`
	}

	err := queryGraphQL(
		`query ($showId: String!) { show( _id: $showId ) { _id availableEpisodesDetail }}`,
		map[string]any{"showId": showID},
		&result,
	)
	
	if eps, ok := result.Data.Show.Available[mode]; ok {
		return eps, err
	}
	return nil, fmt.Errorf("no episodes for mode %s", mode)
}

type Source struct {
	SourceName string `json:"sourceName"`
	SourceUrl  string `json:"sourceUrl"`
}

func GetEpisodeEmbeds(showID, episode, mode string) ([]Source, error) {
	var raw []byte
	err := queryGraphQL(
		`query ($showId: String!, $translationType: VaildTranslationTypeEnumType!, $episodeString: String!) { episode( showId: $showId translationType: $translationType episodeString: $episodeString ) { episodeString sourceUrls }}`,
		map[string]any{"showId": showID, "translationType": mode, "episodeString": episode},
		&raw,
	)
	if err != nil {
		return nil, err
	}

	if start := bytes.Index(raw, []byte(`"tobeparsed":"`)); start != -1 {
		end := bytes.Index(raw[start+14:], []byte(`"`))
		if end != -1 {
			if dec, err := Decrypt(string(raw[start+14 : start+14+end])); err == nil {
				var sources []Source
				json.Unmarshal([]byte(dec), &sources)
				return sources, nil
			}
		}
	}

	var parsed struct {
		Data struct {
			Episode struct {
				SourceUrls []Source `json:"sourceUrls"`
			} `json:"episode"`
		} `json:"data"`
	}
	json.Unmarshal(raw, &parsed)
	return parsed.Data.Episode.SourceUrls, nil
}
