package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/the2dl/linear-term/internal/ui"
)

func main() {
	var (
		tokenFlag    = flag.String("token", "", "Linear Personal API Key (or set LINEAR_API_KEY / ~/.config/linear-term/token)")
		teamFlag     = flag.String("team", "", "Filter by Linear team key prefix (e.g. LIN)")
		intervalFlag = flag.Int("interval", 15, "Live refresh poll interval in seconds")
		demoFlag     = flag.Bool("demo", false, "Run with the sample Delivery board seed data")
		previewFlag  = flag.Bool("preview", false, "Render a single ANSI frame to stdout and exit")
		widthFlag    = flag.Int("width", 168, "Preview width when using --preview")
		heightFlag   = flag.Int("height", 44, "Preview height when using --preview")
	)
	flag.Parse()

	lipgloss.SetColorProfile(termenv.TrueColor)

	cfg := ui.Config{
		Token:        *tokenFlag,
		TeamKey:      *teamFlag,
		PollInterval: time.Duration(*intervalFlag) * time.Second,
		ForceDemo:    *demoFlag,
	}

	model := ui.NewAppModel(cfg)

	if *previewFlag {
		model.SyncOnce()
		updated, _ := model.Update(tea.WindowSizeMsg{Width: *widthFlag, Height: *heightFlag})
		fmt.Println(updated.View())
		return
	}

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running linear-term: %v\n", err)
		os.Exit(1)
	}
}
