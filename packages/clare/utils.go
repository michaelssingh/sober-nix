package main

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	reBr         = regexp.MustCompile(`(?i)<br\s*/?>`)
	reTags       = regexp.MustCompile(`<[^>]*>`)
	reSpaces     = regexp.MustCompile(` {2,}`)
	reBlankLines = regexp.MustCompile(`\n{3,}`)

	reJikanUnits = regexp.MustCompile(`(\d+)\s*(hr|min|sec|s|m|h|hours|hour|minutes|minute|seconds|second)`)
	reDigits     = regexp.MustCompile(`\d+`)
)

func cleanHTML(input string) string {
	if input == "" {
		return ""
	}
	s := html.UnescapeString(input)
	s = reBr.ReplaceAllString(s, "\n")
	s = reTags.ReplaceAllString(s, "")

	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "‘", "'")
	s = strings.ReplaceAll(s, "“", "\"")
	s = strings.ReplaceAll(s, "”", "\"")
	s = strings.ReplaceAll(s, "–", "-")
	s = strings.ReplaceAll(s, "—", "-")
	s = strings.ReplaceAll(s, "\u2026", "...")

	lines := strings.Split(s, "\n")
	for i, line := range lines {
		line = reSpaces.ReplaceAllString(line, " ")
		lines[i] = strings.TrimRight(line, " \t")
	}
	s = strings.Join(lines, "\n")
	s = reBlankLines.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
}

func isMovieMedia(showType, showID, epNo string) bool {
	if strings.EqualFold(showType, "MOVIE") || strings.EqualFold(showType, string(MediaTypeMovie)) {
		return true
	}
	if strings.HasPrefix(showID, "vidsrc:movie") {
		return true
	}
	if strings.EqualFold(epNo, "Movie") || strings.EqualFold(epNo, "Full Movie") {
		return true
	}
	return false
}

func formatMediaTitle(title, epNo string, showTypeAndID ...string) string {
	if title == "" {
		return "Clare Media"
	}
	var showType, showID string
	if len(showTypeAndID) > 0 {
		showType = showTypeAndID[0]
	}
	if len(showTypeAndID) > 1 {
		showID = showTypeAndID[1]
	}
	if isMovieMedia(showType, showID, epNo) {
		return title
	}
	if strings.Contains(title, " - Ep ") || strings.Contains(title, " - S") {
		return title
	}
	if epNo == "" || strings.EqualFold(epNo, "Movie") {
		return title
	}
	cleanEp := epNo
	if strings.HasPrefix(strings.ToLower(cleanEp), "ep ") {
		cleanEp = cleanEp[3:]
	}
	if strings.HasPrefix(strings.ToUpper(cleanEp), "S") && strings.Contains(strings.ToUpper(cleanEp), "E") {
		return fmt.Sprintf("%s - %s", title, cleanEp)
	}
	return fmt.Sprintf("%s - Ep %s", title, cleanEp)
}

func parseEpisodeNumber(ep string) float64 {
	epUpper := strings.ToUpper(strings.TrimSpace(ep))
	if strings.HasPrefix(epUpper, "S") && strings.Contains(epUpper, "E") {
		var s, e int
		if _, err := fmt.Sscanf(epUpper, "S%dE%d", &s, &e); err == nil {
			return float64(s*10000 + e)
		}
	}

	var numStr strings.Builder
	hasDot := false
	for _, r := range ep {
		if r >= '0' && r <= '9' {
			numStr.WriteRune(r)
		} else if r == '.' && !hasDot && numStr.Len() > 0 {
			numStr.WriteRune(r)
			hasDot = true
		} else if numStr.Len() > 0 {
			break
		}
	}
	if numStr.Len() == 0 {
		return 0.0
	}
	val, err := strconv.ParseFloat(numStr.String(), 64)
	if err != nil {
		return 0.0
	}
	return val
}

func parseJikanDuration(d string) float64 {
	if d == "" {
		return 1440.0
	}
	d = strings.ToLower(d)
	total := 0.0
	matches := reJikanUnits.FindAllStringSubmatch(d, -1)
	if len(matches) > 0 {
		for _, m := range matches {
			if val, err := strconv.ParseFloat(m[1], 64); err == nil {
				unit := m[2]
				if strings.HasPrefix(unit, "h") {
					total += val * 3600
				} else if strings.HasPrefix(unit, "m") {
					total += val * 60
				} else if strings.HasPrefix(unit, "s") {
					total += val
				}
			}
		}
		return total
	}
	if m := reDigits.FindString(d); m != "" {
		if val, err := strconv.ParseFloat(m, 64); err == nil {
			return val * 60
		}
	}
	return 1440.0
}

func formatTime(seconds float64) string {
	if seconds <= 0 {
		return "00:00"
	}
	h := int(seconds) / 3600
	m := (int(seconds) % 3600) / 60
	s := int(seconds) % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func stripProviderPrefix(showID string) string {
	if idx := strings.Index(showID, ":"); idx > 0 {
		return showID[idx+1:]
	}
	return showID
}
