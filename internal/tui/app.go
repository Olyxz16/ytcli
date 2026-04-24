package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Olyxz16/ytcli/internal/service"
)

// App is the future TUI application.
type App struct {
	service *service.Service
}

// NewApp creates a new TUI app.
func NewApp(svc *service.Service) *App {
	return &App{service: svc}
}

// Run starts the TUI.
func (a *App) Run() error {
	return fmt.Errorf("TUI not yet implemented")
}

// model is a placeholder bubbletea model.
type model struct {
	msg string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	return m.msg + "\n\nPress 'q' to quit.\n"
}
