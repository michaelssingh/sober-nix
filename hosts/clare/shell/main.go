package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

const (
	host = "0.0.0.0"
	port = "2222" // Use 2222 so we don't conflict with host SSH if running locally
)

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(), // Ensure active terminal
			logging.Middleware(),
		),
	)
	if err != nil {
		slog.Error("Could not start server", "error", err)
		os.Exit(1)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			slog.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	slog.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		slog.Error("Could not stop server", "error", err)
	}
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, active := s.Pty()
	if !active {
		wish.Fatalln(s, "no active terminal, skipping")
		return nil, nil
	}

	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 30

	m := model{
		term:      pty.Term,
		width:     pty.Window.Width,
		height:    pty.Window.Height,
		state:     stateMenu,
		textInput: ti,
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

type sessionState int

const (
	stateMenu sessionState = iota
	stateRegisterUser
	stateRegisterPass
	stateMessage
)

type model struct {
	term      string
	width     int
	height    int
	state     sessionState
	textInput textinput.Model
	username  string
	password  string
	message   string
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		switch m.state {
		case stateMenu:
			switch msg.String() {
			case "1":
				m.state = stateRegisterUser
				m.textInput.Placeholder = "Enter username"
				m.textInput.EchoMode = textinput.EchoNormal
				m.textInput.SetValue("")
				m.textInput.Focus()
				return m, textinput.Blink
			case "2":
				m.message = "Payment logic not implemented yet."
				m.state = stateMessage
				return m, nil
			case "q":
				return m, tea.Quit
			}
		case stateRegisterUser:
			switch msg.String() {
			case "enter":
				m.username = strings.TrimSpace(m.textInput.Value())
				if m.username == "" {
					return m, nil
				}
				m.state = stateRegisterPass
				m.textInput.Placeholder = "Enter password"
				m.textInput.EchoMode = textinput.EchoPassword
				m.textInput.EchoCharacter = '•'
				m.textInput.SetValue("")
				m.textInput.Focus()
				return m, textinput.Blink
			case "esc":
				m.state = stateMenu
				return m, nil
			default:
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		case stateRegisterPass:
			switch msg.String() {
			case "enter":
				m.password = m.textInput.Value()
				if m.password == "" {
					return m, nil
				}
				// Execute sojuctl
				err := createSojuUser(m.username, m.password)
				if err != nil {
					m.message = fmt.Sprintf("Error creating user: %v", err)
				} else {
					m.message = fmt.Sprintf("Successfully created user: %s", m.username)
				}
				m.state = stateMessage
				return m, nil
			case "esc":
				m.state = stateRegisterUser
				m.textInput.Placeholder = "Enter username"
				m.textInput.EchoMode = textinput.EchoNormal
				m.textInput.SetValue(m.username)
				m.textInput.Focus()
				return m, textinput.Blink
			default:
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		case stateMessage:
			switch msg.String() {
			case "enter", "esc", "q":
				m.state = stateMenu
				return m, nil
			}
		}
	}

	return m, cmd
}

func createSojuUser(username, password string) error {
	// Pointing to localhost since they are currently deployed in the same VM, 
	// but this can easily be configured via environment variables for a true HA split.
	client := NewSojuAdminClient("http://localhost:8081")
	return client.CreateUser(username, password)
}

func (m model) View() string {
	var s string
	style := lipgloss.NewStyle().Padding(1, 2)

	switch m.state {
	case stateMenu:
		s = "Welcome to Clare - the Sober Nix IRC Bouncer\n\n"
		s += "1. Register new account\n"
		s += "2. Manage payment\n"
		s += "q. Quit\n"
	case stateRegisterUser:
		s = "Register New Account\n\n"
		s += "Username:\n"
		s += m.textInput.View() + "\n\n"
		s += "(Press Enter to continue, Esc to cancel)"
	case stateRegisterPass:
		s = "Register New Account\n\n"
		s += "Password:\n"
		s += m.textInput.View() + "\n\n"
		s += "(Press Enter to create account, Esc to go back)"
	case stateMessage:
		s = m.message + "\n\n"
		s += "(Press Enter to return to menu)"
	}

	return style.Render(s)
}
