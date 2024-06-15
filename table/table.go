package table

import (
	"fmt"
	"strings"
	"time"

	t "tsqlr/tests"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type TickMsg time.Time

type Mode int

const (
	TABLE Mode = iota
	VIEWPORT
	TEXTAREA
	// HELP
)

type Model struct {
	Tests    []t.Test
	queue    chan *t.Test
	table    table.Model
	viewport viewport.Model
	textarea textarea.Model
	mode     Mode
	chosen   *t.Test
	updating bool
}

func testToRow(test t.Test) table.Row {
	return table.Row{test.Status.String(), test.String()}
}

func buildRows(tests []t.Test) []table.Row {
	rows := []table.Row{}
	for _, test := range tests {
		rows = append(rows, testToRow(test))
	}
	return rows
}

func (m *Model) runTest(i int) {
	test := &m.Tests[i]
	if test.Status == t.RUNNING {
		return
	}

	test.Status = t.RUNNING
	m.queue <- test
}

func (m *Model) RemoveTest(i int) bool {
	// remove a m.Tests[i] and recreate m.Tests without gaps
	if i < 0 || i >= len(m.Tests) {
		return false
	}

	var tests []t.Test
	for j := range m.Tests {
		if j != i {
			tests = append(tests, m.Tests[j])
		}
	}

	m.Tests = tests
	return true
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) UpdateTable(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			m.mode = VIEWPORT
			m.chosen = &m.Tests[m.table.Cursor()]
			return m.UpdateViewport("Open")
		case "r": // rerun the selected test
			m.runTest(m.table.Cursor())
			return m.UpdateTable("TestUpdated")
		case "R": // rerun all tests
			go func() {
				for i := range m.Tests {
					m.runTest(i)
				}
			}()
			return m.UpdateTable("TestUpdated")
		case "d", "x":
			cursor := m.table.Cursor()
			if ok := m.RemoveTest(cursor); ok {
				m.table.SetRows(buildRows(m.Tests))
				m.table.UpdateViewport()
				if cursor > 0 {
					m.table.SetCursor(cursor - 1)
				}
				// update table but prevent default keypress event
				m.table, cmd = m.table.Update(nil)
				return m, cmd
			}
		}
	case TickMsg:
		m.updating = false
		return m.UpdateTable("TestUpdated")
	case tea.Msg:
		switch msg := msg.(type) {
		case string:
			switch msg {
			case "Open":
				m.updating = false
				m.table.SetRows(buildRows(m.Tests))
				m.table.UpdateViewport()
				return m, nil
			case "TestUpdated":
				if m.updating == true {
					return m, nil
				}

				m.updating = true
				m.table.SetRows(buildRows(m.Tests))
				m.table.UpdateViewport()

				return m, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
					return TickMsg(t)
				})
			}
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) UpdateViewport(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.mode = TABLE
			m.chosen = nil
			return m.UpdateTable("Open")
		case "r": // rerun the selected test
			m.runTest(m.table.Cursor())
			return m.UpdateViewport("TestUpdated")
		case "j", "down": // move to next test
			cursor := m.table.Cursor()
			if cursor < len(m.Tests)-1 {
				cursor = cursor + 1
				m.table.SetCursor(cursor)
				m.chosen = &m.Tests[cursor]
			}
			return m.UpdateViewport("Open")
		case "k", "up": // move to prev test
			cursor := m.table.Cursor()
			if cursor > 0 {
				cursor = cursor - 1
				m.table.SetCursor(cursor)
				m.chosen = &m.Tests[cursor]
			}
			return m.UpdateViewport("Open")
		}
	case tea.Msg:
		switch msg := msg.(type) {
		case string:
			switch msg {
			case "Open", "TestUpdated":
				var content string
				chosen := *m.chosen
				switch chosen.Status {
				case t.RUNNING:
					content = "Test running..."
				default:
					content = strings.Join(chosen.Results, "\n")
				}
				m.viewport.SetContent(content)
				return m, nil
			}
		}
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) UpdateTextarea(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	// switch msg := msg.(type) {
	// case tea.KeyMsg:
	// 	switch msg.String() {
	// 	case "esc":
	// 		return m, tea.Quit
	// 	}
	// }
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height - 4)
		m.table.SetColumns([]table.Column{
			{Title: "Status", Width: 8},
			{Title: "Test/Suite", Width: msg.Width - 8},
		})
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 6
	}

	switch m.mode {
	case TEXTAREA:
		return m.UpdateTextarea(msg)
	case VIEWPORT:
		return m.UpdateViewport(msg)
	case TABLE:
		fallthrough
	default:
		return m.UpdateTable(msg)
	}
}

func statusColor(s t.Status) lipgloss.Style {
	switch s {
	case t.RUNNING: // yellow
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFF000"))
	case t.PASS: // green
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF00"))
	case t.FAIL: // red
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF0000"))
	case t.ERROR: // orange
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF8000"))
	default:
		return lipgloss.NewStyle().Bold(false)
	}
}

func (m Model) viewportTitle() string {
	if m.chosen == nil {
		return ""
	}
	titleStyle := func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()
	test := m.chosen
	title := titleStyle.Render(fmt.Sprintf("%s | %s",
		statusColor(test.Status).Render(test.Status.String()),
		test.String()))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m Model) View() string {
	var view string
	switch {
	case m.chosen != nil:
		view = fmt.Sprintf("%s\n%s", m.viewportTitle(), m.viewport.View())
	default:
		view = m.table.View()
	}
	return baseStyle.Render(view) + "\n"
}

func InitialModel(queue chan *t.Test, tests []t.Test) Model {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "Status"},
			{Title: "Test/Suite"},
		}),
		table.WithRows(buildRows(tests)),
		table.WithFocused(true),
		table.WithStyleFunc(func(row, col int, s string) lipgloss.Style {
			if col == 0 { // status column
				switch s {
				case t.RUNNING.String():
					return statusColor(t.RUNNING)
				case t.PASS.String():
					return statusColor(t.PASS)
				case t.FAIL.String():
					return statusColor(t.FAIL)
				case t.ERROR.String():
					return statusColor(t.ERROR)
				default:
					return lipgloss.NewStyle().Bold(false)
				}
			}
			return lipgloss.NewStyle().Bold(false)
		}),
	)
	t.SetStyles(s)

	return Model{
		Tests:    tests,
		queue:    queue,
		table:    t,
		viewport: viewport.New(t.Width(), t.Height()),
		textarea: textarea.New(),
		mode:     TABLE,
	}
}
