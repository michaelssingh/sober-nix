package main

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type EventDomain string

const (
	DomainSearch   EventDomain = "SEARCH"
	DomainResolve  EventDomain = "RESOLVE"
	DomainAniSkip  EventDomain = "ANISKIP"
	DomainMpvIPC   EventDomain = "MPV_IPC"
	DomainPosition EventDomain = "POSITION"
	DomainTUI      EventDomain = "TUI"
)

var (
	Logger     *slog.Logger
	LogFile    *os.File
	loggerOnce sync.Once
)

type LogEvent struct {
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Domain  EventDomain    `json:"domain"`
	Message string         `json:"msg"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

func InitLogger(logPath string) error {
	var err error
	loggerOnce.Do(func() {
		if logPath == "" {
			home, _ := os.UserHomeDir()
			logPath = filepath.Join(home, ".local", "state", "clare", "debug.log")
		}
		_ = os.MkdirAll(filepath.Dir(logPath), 0755)
		LogFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return
		}
		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
		handler := slog.NewJSONHandler(LogFile, opts)
		Logger = slog.New(handler)
	})
	return err
}

func LogEventInfo(domain EventDomain, msg string, attrs ...slog.Attr) {
	if Logger == nil {
		_ = InitLogger("")
	}
	args := make([]any, 0, len(attrs)+1)
	args = append(args, slog.String("domain", string(domain)))
	for _, a := range attrs {
		args = append(args, a)
	}
	Logger.Info(msg, args...)
}

func LogEventError(domain EventDomain, msg string, err error, attrs ...slog.Attr) {
	if Logger == nil {
		_ = InitLogger("")
	}
	args := make([]any, 0, len(attrs)+2)
	args = append(args, slog.String("domain", string(domain)))
	if err != nil {
		args = append(args, slog.String("error", err.Error()))
	}
	for _, a := range attrs {
		args = append(args, a)
	}
	Logger.Error(msg, args...)
}

type HealthSummary struct {
	SearchSuccess  bool     `json:"search_success"`
	PreflightOK    bool     `json:"preflight_ok"`
	AniSkipFound   bool     `json:"aniskip_found"`
	VideoCodecOK   bool     `json:"video_codec_ok"`
	AudioCodecOK   bool     `json:"audio_codec_ok"`
	PlaybackErrors []string `json:"playback_errors"`
	PositionSaved  bool     `json:"position_saved"`
	OverallStatus  string   `json:"overall_status"` // "OPTIMAL", "SUBOPTIMAL", "FAILED"
}

func ValidateSessionTrace(logPath string) (HealthSummary, error) {
	if logPath == "" {
		home, _ := os.UserHomeDir()
		logPath = filepath.Join(home, ".local", "state", "clare", "debug.log")
	}
	file, err := os.Open(logPath)
	if err != nil {
		return HealthSummary{OverallStatus: "FAILED"}, err
	}
	defer file.Close()

	summary := HealthSummary{OverallStatus: "OPTIMAL"}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var evt LogEvent
		line := scanner.Bytes()
		if err := json.Unmarshal(line, &evt); err != nil {
			// Fallback text matching for non-JSON legacy logs
			strLine := string(line)
			if strings.Contains(strLine, "[INFO] --- Automated Test Playback Started") || strings.Contains(strLine, "[INFO] --- Playback Started") {
				summary.SearchSuccess = true
			}
			if strings.Contains(strLine, "HTTP error 403") || strings.Contains(strLine, "Errors when loading file") {
				summary.PlaybackErrors = append(summary.PlaybackErrors, strLine)
			}
			continue
		}

		switch evt.Domain {
		case DomainSearch:
			if evt.Level == "INFO" && (strings.Contains(evt.Message, "search results") || strings.Contains(evt.Message, "Selected Show")) {
				summary.SearchSuccess = true
			}
		case DomainResolve:
			if evt.Level == "INFO" && strings.Contains(evt.Message, "preflight success") {
				summary.PreflightOK = true
			}
		case DomainAniSkip:
			if evt.Level == "INFO" && strings.Contains(evt.Message, "skip times found") {
				summary.AniSkipFound = true
			}
		case DomainMpvIPC:
			if strings.Contains(evt.Message, "video codec initialized") || strings.Contains(evt.Message, "video_codec") {
				summary.VideoCodecOK = true
			}
			if strings.Contains(evt.Message, "audio codec initialized") || strings.Contains(evt.Message, "audio_codec") {
				summary.AudioCodecOK = true
			}
			if evt.Level == "ERROR" || strings.Contains(evt.Message, "HTTP error 403") || strings.Contains(evt.Message, "Failed to open") {
				summary.PlaybackErrors = append(summary.PlaybackErrors, evt.Message)
			}
		case DomainPosition:
			if strings.Contains(evt.Message, "saved position") {
				summary.PositionSaved = true
			}
		}
	}

	if len(summary.PlaybackErrors) > 0 || !summary.PreflightOK || !summary.VideoCodecOK {
		summary.OverallStatus = "FAILED"
	} else if !summary.AniSkipFound || !summary.PositionSaved {
		summary.OverallStatus = "SUBOPTIMAL"
	}

	return summary, nil
}
