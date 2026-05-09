package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hamza-hafeez82/cortex/pkg/detector"
)

// Stage represents one of the three scan stages.
type Stage int

const (
	StageIdle Stage = iota
	StageRecon
	StageSecurity
	StageArchitecture
	StageDone
)

// StageUpdate is sent to the TUI when a stage completes.
type StageUpdate struct {
	Stage  Stage
	Issues []detector.Issue
	Done   bool
	Err    error
}

// Styles
var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D4FF"))
	styleActive   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4FF"))
	styleDone     = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88"))
	stylePending  = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	styleIssue    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00"))
	styleCrit     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF4444"))
	styleFilePath = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	styleDimTUI   = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
)

// Model is the Bubbletea model for the scan progress view.
type Model struct {
	spinner      spinner.Model
	stage        Stage
	repoPath     string
	totalFiles   int
	issues       []detector.Issue
	reconDone    bool
	securityDone bool
	archDone     bool
	secCount     int
	archCount    int
	depCount     int
	startTime    time.Time
	updateChan   chan StageUpdate
	done         bool
	err          error
}

// NewModel creates a scan TUI model.
func NewModel(repoPath string, totalFiles int, updateChan chan StageUpdate) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styleActive

	return Model{
		spinner:    sp,
		repoPath:   repoPath,
		totalFiles: totalFiles,
		updateChan: updateChan,
		startTime:  time.Now(),
	}
}

// waitForUpdate is a tea.Cmd that reads the next stage update.
func waitForUpdate(ch chan StageUpdate) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, waitForUpdate(m.updateChan))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case StageUpdate:
		m.issues = append(m.issues, msg.Issues...)

		for _, issue := range msg.Issues {
			switch issue.Category {
			case detector.CategorySecurity, detector.CategoryDependency:
				if issue.Category == detector.CategoryDependency {
					m.depCount++
				} else {
					m.secCount++
				}
			case detector.CategoryArchitecture:
				m.archCount++
			}
		}

		switch msg.Stage {
		case StageRecon:
			m.reconDone = true
			m.stage = StageSecurity
		case StageSecurity:
			m.securityDone = true
			m.stage = StageArchitecture
		case StageArchitecture:
			m.archDone = true
			m.stage = StageDone
			m.done = true
		}

		if msg.Err != nil {
			m.err = msg.Err
			m.done = true
			return m, tea.Quit
		}

		if msg.Done {
			m.done = true
			return m, tea.Quit
		}

		return m, tea.Batch(m.spinner.Tick, waitForUpdate(m.updateChan))
	}

	return m, nil
}

func (m Model) View() string {
	var sb strings.Builder

	// Header
	sb.WriteString("\n")
	sb.WriteString(styleTitle.Render("  CORTEX") + "  " +
		styleDimTUI.Render(fmt.Sprintf("scanning %s", m.repoPath)) + "\n\n")

	// Stage 1 — Recon
	sb.WriteString(stageRow(
		"Stage 1", "Reconnaissance",
		m.reconDone, m.stage == StageRecon,
		m.spinner,
		"",
	))

	// Stage 2 — Security
	secDetail := ""
	if m.securityDone {
		secDetail = issueCount(m.secCount+m.depCount, "issue")
	}
	sb.WriteString(stageRow(
		"Stage 2", "Security",
		m.securityDone, m.stage == StageSecurity,
		m.spinner,
		secDetail,
	))

	// Stage 3 — Architecture
	archDetail := ""
	if m.archDone {
		archDetail = issueCount(m.archCount, "issue")
	}
	sb.WriteString(stageRow(
		"Stage 3", "Architecture",
		m.archDone, m.stage == StageArchitecture,
		m.spinner,
		archDetail,
	))

	sb.WriteString("\n")

	// Live findings feed (last 5 issues)
	if len(m.issues) > 0 {
		sb.WriteString(styleDimTUI.Render("  Recent findings:") + "\n")
		start := len(m.issues) - 5
		if start < 0 {
			start = 0
		}
		for _, issue := range m.issues[start:] {
			sev := issue.Severity
			sevStr := ""
			switch sev {
			case detector.SeverityCritical:
				sevStr = styleCrit.Render("CRIT")
			case detector.SeverityHigh:
				sevStr = styleIssue.Render("HIGH")
			default:
				sevStr = styleDimTUI.Render(strings.ToUpper(sev[:3]))
			}
			loc := issue.File
			if issue.Line > 0 {
				loc = fmt.Sprintf("%s:%d", issue.File, issue.Line)
			}
			sb.WriteString(fmt.Sprintf("  %s  %s  %s\n",
				sevStr,
				styleActive.Render(issue.Code),
				styleFilePath.Render(loc),
			))
		}
		sb.WriteString("\n")
	}

	if m.done {
		elapsed := time.Since(m.startTime).Round(time.Millisecond)
		if m.err != nil {
			sb.WriteString(fmt.Sprintf("  %s  %s\n\n",
				styleCrit.Render("✖ Scan failed"),
				styleDimTUI.Render(m.err.Error()),
			))
		} else {
			total := m.secCount + m.depCount + m.archCount
			sb.WriteString(fmt.Sprintf("  %s  %s  %s\n\n",
				styleDone.Render("✓ Scan complete"),
				styleDimTUI.Render(fmt.Sprintf("%d issues", total)),
				styleDimTUI.Render(fmt.Sprintf("(%s)", elapsed)),
			))
		}
	}

	return sb.String()
}

func stageRow(num, name string, done, active bool, sp spinner.Model, detail string) string {
	var icon, stageStyle string

	if done {
		icon = styleDone.Render("✓")
		stageStyle = styleDone.Render(fmt.Sprintf("%-8s  %s", num, name))
	} else if active {
		icon = sp.View()
		stageStyle = styleActive.Render(fmt.Sprintf("%-8s  %s", num, name))
	} else {
		icon = stylePending.Render("○")
		stageStyle = stylePending.Render(fmt.Sprintf("%-8s  %s", num, name))
	}

	row := fmt.Sprintf("  %s  %s", icon, stageStyle)
	if detail != "" {
		row += "  " + styleDimTUI.Render(detail)
	}
	return row + "\n"
}

func issueCount(n int, unit string) string {
	if n == 0 {
		return "clean"
	}
	return fmt.Sprintf("%d %s%s", n, unit, plural(n))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
