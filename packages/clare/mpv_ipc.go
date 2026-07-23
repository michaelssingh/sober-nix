package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"time"
)

type MPVIPCClient struct {
	socketPath string
	conn       net.Conn
	requestID  int
}

type MPVIPCRequest struct {
	Command   []any `json:"command"`
	RequestID int   `json:"request_id"`
}

type MPVIPCResponse struct {
	Error     string `json:"error"`
	Data      any    `json:"data"`
	RequestID int    `json:"request_id"`
	Event     string `json:"event"`
}

func NewMPVIPCClient(socketPath string) (*MPVIPCClient, error) {
	conn, err := net.DialTimeout("unix", socketPath, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MPV IPC socket %s: %w", socketPath, err)
	}
	return &MPVIPCClient{socketPath: socketPath, conn: conn}, nil
}

func (c *MPVIPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *MPVIPCClient) GetProperty(prop string) (any, error) {
	c.requestID++
	req := MPVIPCRequest{
		Command:   []any{"get_property", prop},
		RequestID: c.requestID,
	}
	payload, _ := json.Marshal(req)
	payload = append(payload, '\n')

	_ = c.conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := c.conn.Write(payload); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(c.conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		var resp MPVIPCResponse
		if err := json.Unmarshal(line, &resp); err == nil {
			if resp.RequestID == c.requestID {
				if resp.Error != "success" {
					return nil, fmt.Errorf("mpv ipc error: %s", resp.Error)
				}
				return resp.Data, nil
			}
		}
	}
}

func (c *MPVIPCClient) Seek(seconds float64) error {
	c.requestID++
	req := MPVIPCRequest{
		Command:   []any{"seek", seconds, "relative"},
		RequestID: c.requestID,
	}
	payload, _ := json.Marshal(req)
	payload = append(payload, '\n')

	_ = c.conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := c.conn.Write(payload); err != nil {
		return err
	}
	return nil
}

type VideoHealthStats struct {
	VideoCodec  string  `json:"video_codec"`
	AudioCodec  string  `json:"audio_codec"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	PlaybackPos float64 `json:"playback_pos"`
}

func (c *MPVIPCClient) InspectHealth() (VideoHealthStats, error) {
	stats := VideoHealthStats{}
	if v, err := c.GetProperty("video-format"); err == nil && v != nil {
		stats.VideoCodec = fmt.Sprintf("%v", v)
		LogEventInfo(DomainMpvIPC, "video codec initialized", slog.String("codec", stats.VideoCodec))
	}
	if a, err := c.GetProperty("audio-codec-name"); err == nil && a != nil {
		stats.AudioCodec = fmt.Sprintf("%v", a)
		LogEventInfo(DomainMpvIPC, "audio codec initialized", slog.String("codec", stats.AudioCodec))
	}
	if w, err := c.GetProperty("width"); err == nil && w != nil {
		if wf, ok := w.(float64); ok {
			stats.Width = int(wf)
		}
	}
	if h, err := c.GetProperty("height"); err == nil && h != nil {
		if hf, ok := h.(float64); ok {
			stats.Height = int(hf)
		}
	}
	if p, err := c.GetProperty("playback-time"); err == nil && p != nil {
		if pf, ok := p.(float64); ok {
			stats.PlaybackPos = pf
		}
	}
	return stats, nil
}
