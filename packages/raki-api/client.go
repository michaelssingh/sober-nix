package adminapi

import (
	"context"
	"fmt"
	"gopkg.in/irc.v4"
	"net"
	"strconv"
	"strings"
	"time"
)

// dialAdmin connects to the Soju unix socket.
func (s *Server) dialAdmin(ctx context.Context) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, "unix", s.adminSocketPath)
}

// sendAdminCommand sends a command to the Soju admin socket and returns the response.
func (s *Server) sendAdminCommand(ctx context.Context, words []string) (string, error) {
	uc, err := s.dialAdmin(ctx)
	if err != nil {
		return "", fmt.Errorf("dial: %v", err)
	}
	defer uc.Close()
	uc.SetDeadline(time.Now().Add(5 * time.Second))

	c := irc.NewConn(uc)
	if err := c.WriteMessage(&irc.Message{
		Command: "BOUNCERSERV",
		Params:  []string{quoteWords(words)},
	}); err != nil {
		return "", fmt.Errorf("write: %v", err)
	}

	var output strings.Builder
	for {
		m, err := c.ReadMessage()
		if err != nil {
			return "", fmt.Errorf("read: %v", err)
		}
		switch m.Command {
		case "PRIVMSG":
			if output.Len() > 0 {
				output.WriteRune('\n')
			}
			output.WriteString(m.Trailing())
		case "BOUNCERSERV":
			if m.Param(0) == "OK" {
				return output.String(), nil
			}
			// If not OK, it's an error
			return "", fmt.Errorf("error: %s", m.Trailing())
		case "ERROR":
			return "", fmt.Errorf("error: %s", m.Trailing())
		}
	}
}

func quoteWords(words []string) string {
	var s strings.Builder
	for _, word := range words {
		if s.Len() > 0 {
			s.WriteRune(' ')
		}
		s.WriteString(strconv.Quote(word))
	}
	return s.String()
}
