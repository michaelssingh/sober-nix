package main

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

func cleanHTML(input string) string {
	if input == "" {
		return ""
	}
	s := html.UnescapeString(input)
	reBr := regexp.MustCompile(`(?i)<br\s*/?>`)
	s = reBr.ReplaceAllString(s, "\n")
	reTags := regexp.MustCompile(`<[^>]*>`)
	s = reTags.ReplaceAllString(s, "")

	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "‘", "'")
	s = strings.ReplaceAll(s, "“", "\"")
	s = strings.ReplaceAll(s, "”", "\"")
	s = strings.ReplaceAll(s, "–", "-")
	s = strings.ReplaceAll(s, "—", "-")
	s = strings.ReplaceAll(s, "\u2026", "...")

	reSpaces := regexp.MustCompile(` {2,}`)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		line = reSpaces.ReplaceAllString(line, " ")
		lines[i] = strings.TrimRight(line, " \t")
	}
	s = strings.Join(lines, "\n")

	reBlankLines := regexp.MustCompile(`\n{3,}`)
	s = reBlankLines.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
}

func formatMediaTitle(title, epNo string) string {
	if title == "" {
		return "Clare Media"
	}
	if epNo == "" || strings.EqualFold(epNo, "Movie") || strings.EqualFold(epNo, "1") {
		return title
	}
	if strings.HasPrefix(strings.ToUpper(epNo), "S") && strings.Contains(strings.ToUpper(epNo), "E") {
		return fmt.Sprintf("%s - %s", title, epNo)
	}
	return fmt.Sprintf("%s - Episode %s", title, epNo)
}

func parseEpisodeNumber(ep string) float64 {
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
	r := regexp.MustCompile(`(\d+)\s*(hr|min|sec|s|m|h|hours|hour|minutes|minute|seconds|second)`)
	matches := r.FindAllStringSubmatch(d, -1)
	if len(matches) > 0 {
		for _, m := range matches {
			var val float64
			fmt.Sscanf(m[1], "%f", &val)
			unit := m[2]
			if strings.HasPrefix(unit, "h") {
				total += val * 3600
			} else if strings.HasPrefix(unit, "m") {
				total += val * 60
			} else if strings.HasPrefix(unit, "s") {
				total += val
			}
		}
		return total
	}
	rDigits := regexp.MustCompile(`\d+`)
	if m := rDigits.FindString(d); m != "" {
		var val float64
		fmt.Sscanf(m, "%f", &val)
		return val * 60
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
