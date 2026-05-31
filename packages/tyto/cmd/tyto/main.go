package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/sober-nix/tyto"
)

var decodeMap = map[string]string{
	"79": "A", "7a": "B", "7b": "C", "7c": "D", "7d": "E", "7e": "F", "7f": "G",
	"70": "H", "71": "I", "72": "J", "73": "K", "74": "L", "75": "M", "76": "N", "77": "O",
	"68": "P", "69": "Q", "6a": "R", "6b": "S", "6c": "T", "6d": "U", "6e": "V", "6f": "W",
	"60": "X", "61": "Y", "62": "Z", "59": "a", "5a": "b", "5b": "c", "5c": "d", "5d": "e",
	"5e": "f", "5f": "g", "50": "h", "51": "i", "52": "j", "53": "k", "54": "l", "55": "m",
	"56": "n", "57": "o", "48": "p", "49": "q", "4a": "r", "4b": "s", "4c": "t", "4d": "u",
	"4e": "v", "4f": "w", "40": "x", "41": "y", "42": "z", "08": "0", "09": "1", "0a": "2",
	"0b": "3", "0c": "4", "0d": "5", "0e": "6", "0f": "7", "00": "8", "01": "9", "15": "-",
	"16": ".", "67": "_", "46": "~", "02": ":", "17": "/", "07": "?", "1b": "#", "63": "[",
	"65": "]", "78": "@", "19": "!", "1c": "$", "1e": "&", "10": "(", "11": ")", "12": "*",
	"13": "+", "14": ",", "03": ";", "05": "=", "1d": "%",
}

func decodeDashURL(url string) string {
	url = strings.TrimPrefix(url, "--")
	var sb strings.Builder
	for i := 0; i < len(url); i += 2 {
		if i+2 > len(url) {
			break
		}
		pair := url[i : i+2]
		if char, ok := decodeMap[pair]; ok {
			sb.WriteString(char)
		} else {
			sb.WriteString(pair)
		}
	}
	return strings.ReplaceAll(sb.String(), "/clock", "/clock.json")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tyto <search query>")
		os.Exit(1)
	}

	query := strings.Join(os.Args[1:], " ")
	mode := "sub"

	fmt.Printf("Searching for %q...\n", query)
	animes, err := tyto.SearchAnime(query, mode)
	if err != nil || len(animes) == 0 {
		fmt.Println("No results found.")
		return
	}

	idx, err := fuzzyfinder.Find(animes, func(i int) string { return animes[i].Name })
	if err != nil {
		return
	}
	anime := animes[idx]

	episodes, err := tyto.GetEpisodes(anime.ID, mode)
	if err != nil || len(episodes) == 0 {
		fmt.Println("No episodes available.")
		return
	}

	epIdx, err := fuzzyfinder.Find(episodes, func(i int) string { return "Episode " + episodes[i] })
	if err != nil {
		return
	}
	episode := episodes[epIdx]

	embeds, err := tyto.GetEpisodeEmbeds(anime.ID, episode, mode)
	if err != nil || len(embeds) == 0 {
		fmt.Println("No streams found.")
		return
	}

	embedIdx, err := fuzzyfinder.Find(embeds, func(i int) string { return embeds[i].SourceName })
	if err != nil {
		return
	}
	src := embeds[embedIdx]

	url := src.SourceUrl
	if strings.HasPrefix(url, "--") {
		url = decodeDashURL(url)
	}
	if strings.HasPrefix(url, "/") {
		url = "https://allanime.day" + url
	}

	fmt.Printf("Playing: %s\n", url)
	cmd := exec.Command("mpv", "--tls-verify=no", url)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.Run()
}
