package tui

import (
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// You generally won't need this unless you're processing stuff with
// complicated ANSI escape sequences. Turn it on if you notice flickering.
//
// Also keep in mind that high performance rendering only works for programs
// that use the full size of the terminal. We're enabling that below with
// tea.EnterAltScreen().
const useHighPerformanceRenderer = false

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()
)

type LogTick time.Time

func logTicker() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return LogTick(t) })
}

type LogsPagerModel struct {
	containerId string
	rc          io.ReadCloser
	content     *strings.Builder
	ready       bool
	viewport    viewport.Model
}

func (m LogsPagerModel) Init() tea.Cmd {
	return logTicker()
}

func (m LogsPagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Println("from pager", reflect.TypeOf(msg))
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(moreInfoStyle.GetWidth(), moreInfoStyle.GetHeight()-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer

			// read from m.rc and set viewport
			if m.rc != nil {
				m.readMoreLogs()
			}
			m.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		if useHighPerformanceRenderer {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(m.viewport))
		}
	case LogTick:
		if m.rc != nil {
			m.readMoreLogs()
		}
		cmds = append(cmds, logTicker())
	}

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m LogsPagerModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	// return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
	return fmt.Sprintf("%s\n%s\n%s", "", m.viewport.View(), "")
}

func (m LogsPagerModel) headerView() string {
	title := titleStyle.Render("Mr. Pager")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m LogsPagerModel) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m *LogsPagerModel) setCurrentContainerId(id string) {
	log.Println("settings containerid")
	m.containerId = id
}

func (m *LogsPagerModel) setReaderCloser(rc io.ReadCloser) {
	log.Println("settings rc")
	if m.rc != nil {
		m.rc.Close()
	}
	m.rc = rc
}

func (m *LogsPagerModel) readMoreLogs() {
	for {
		buffer := make([]byte, 100)
		bytesRead, _ := m.rc.Read(buffer)
		if bytesRead == 0 {
			log.Println("eof reached")
			m.viewport.SetContent(m.content.String())
			return
		}

		m.content.Write(buffer)
	}

}

// func main() {
// 	// Load some text for our viewport
// 	content, err := os.ReadFile("artichoke.md")
// 	if err != nil {
// 		fmt.Println("could not load file:", err)
// 		os.Exit(1)
// 	}

// 	p := tea.NewProgram(
// 		,
// 		tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
// 		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
// 	)

// 	if _, err := p.Run(); err != nil {
// 		fmt.Println("could not run program:", err)
// 		os.Exit(1)
// 	}
// }
