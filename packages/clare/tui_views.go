package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderShowDetailsPanel(show AnimeShow, coverArtAnsi string, width, height int) string {
	if width < 10 {
		width = 10
	}
	if height < 5 {
		height = 5
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7aa2f7")).
		Padding(0, 1).
		Width(width).
		Height(height)

	var b strings.Builder

	b.WriteString(headerStyle.Render("◆ SHOW DETAILS ◆") + "\n")
	b.WriteString(panelTitleStyle.Render(show.Name) + "\n")
	if show.EnglishName != "" && show.EnglishName != show.Name {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Italic(true).Render(show.EnglishName) + "\n")
	}
	b.WriteString("\n")

	if show.Score > 0 {
		b.WriteString(fmt.Sprintf("%s  %.2f\n", accentColorStyle.Render("Score:"), show.Score))
	}
	if show.Type != "" {
		b.WriteString(fmt.Sprintf("%s  %s\n", accentColorStyle.Render("Format:"), strings.ToUpper(show.Type)))
	}
	if show.Season.Year > 0 {
		b.WriteString(fmt.Sprintf("%s  %d\n", accentColorStyle.Render("Release:"), show.Season.Year))
	}

	epCount := show.EpCount()
	if epCount > 0 {
		durationStr := ""
		if show.Duration != "" {
			durationStr = fmt.Sprintf(" × %s", show.Duration)
		}
		b.WriteString(fmt.Sprintf("%s  %d eps%s\n", accentColorStyle.Render("Length:"), epCount, durationStr))
	}

	if len(show.Genres) > 0 {
		b.WriteString(fmt.Sprintf("%s  %s\n", accentColorStyle.Render("Genres:"), strings.Join(show.Genres, ", ")))
	}

	b.WriteString("\n")

	synopsis := "No description available."
	if show.Description != "" {
		synopsis = cleanHTML(show.Description)
	}

	synLines := strings.Split(synBodyStyle.Render(synopsis), "\n")
	currLine := m.detailsScrollOffset + 1
	maxLine := len(synLines)
	if maxLine < 1 {
		maxLine = 1
	}

	synopsisHeader := "◆ SHOW OVERVIEW ◆"
	if maxLine > 1 {
		synopsisHeader = fmt.Sprintf("◆ SHOW OVERVIEW (scroll: h/l) [%d/%d] ◆", currLine, maxLine)
	}

	b.WriteString(headerStyle.Render(synopsisHeader) + "\n")
	startLine := m.detailsScrollOffset
	if startLine >= len(synLines) {
		startLine = len(synLines) - 1
	}
	if startLine < 0 {
		startLine = 0
	}
	visibleSynLines := synLines[startLine:]
	b.WriteString(strings.Join(visibleSynLines, "\n"))

	content := b.String()
	return borderStyle.Render(content)
}

func (m model) renderErrorModal(baseView string) string {
	if m.errorPopupMsg == "" {
		return baseView
	}

	boxWidth := 58
	if m.width > 10 && boxWidth > m.width-4 {
		boxWidth = m.width - 4
	}
	if boxWidth < 20 {
		boxWidth = 20
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#f7768e")).
		Align(lipgloss.Center).
		Width(boxWidth - 4)

	bodyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#c0caf5")).
		Align(lipgloss.Center).
		Width(boxWidth - 4)

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#565f89")).
		Italic(true).
		Align(lipgloss.Center).
		Width(boxWidth - 4)

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#f7768e")).
		Background(lipgloss.Color("#1a1b26")).
		Padding(1, 2).
		Width(boxWidth).
		Render(fmt.Sprintf("%s\n\n%s\n\n%s",
			headerStyle.Render("⚠️ ERROR"),
			bodyStyle.Render(m.errorPopupMsg),
			footerStyle.Render("Press [Enter] or [Esc] to dismiss"),
		))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalBox, lipgloss.WithWhitespaceChars(" "))
}
