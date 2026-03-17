package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sessionState int

const (
	stateNav sessionState = iota
	stateEdit
)

var (
	// --- The Zen Palette Refined ---
	sand       = lipgloss.Color("#D8E2DC") // Creamy soft text
	evergreen  = lipgloss.Color("#2F3E46") // Deep sidebar
	charcoal   = lipgloss.Color("#1B262C") // Darker, softer main background
	leaf       = lipgloss.Color("#84A59D") // Muted green for accents
	softGold   = lipgloss.Color("#E9C46A") // Warm selection highlight

	// --- Styles ---
	sidebarStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Background(evergreen).
			Foreground(sand)

	contentStyle = lipgloss.NewStyle().
			Padding(2, 4).
			Background(charcoal)

	activeTabStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(leaf).
			Foreground(leaf).
			Padding(0, 1).
			Italic(true)

	listSelected = lipgloss.NewStyle().
			Foreground(softGold).
			Bold(true)

	instructionStyle = lipgloss.NewStyle().Foreground(leaf).Faint(true)
	titleStyle       = lipgloss.NewStyle().Foreground(leaf).Bold(true).Underline(true)
)

type journalEntry struct {
	filename string
	title    string
	date     string
	content  string
}

type model struct {
	state        sessionState
	editor       textarea.Model
	viewPort     viewport.Model
	journals     []journalEntry
	cursor       int
	width        int
	height       int
	ready        bool
	sidebarWidth int
	activeFile   string
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Write your heart out..."
	ta.Focus()
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(sand)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle() 
	ta.Cursor.Style = lipgloss.NewStyle().Foreground(leaf)

	return model{
		state:        stateNav,
		editor:       ta,
		sidebarWidth: 35,
	}
}

func (m model) Init() tea.Cmd { return textarea.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentWidth := m.width - m.sidebarWidth - 10
		
		if !m.ready {
			m.viewPort = viewport.New(contentWidth, m.height-12)
			m.ready = true
		} else {
			m.viewPort.Width = contentWidth
			m.viewPort.Height = m.height - 12
		}
		m.editor.SetWidth(contentWidth)
		m.editor.SetHeight(m.height - 12)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		switch m.state {
		case stateNav:
			switch msg.String() {
			case "n":
				m.editor.Reset()
				m.activeFile = ""
				m.state = stateEdit
				m.editor.Focus()
			case "up", "k":
				if m.cursor > 0 { m.cursor-- }
				m.syncView()
			case "down", "j":
				if m.cursor < len(m.journals)-1 { m.cursor++ }
				m.syncView()
			case "enter", "e":
				if len(m.journals) > 0 {
					m.activeFile = m.journals[m.cursor].filename
					m.editor.SetValue(m.journals[m.cursor].content)
					m.state = stateEdit
					m.editor.Focus()
				}
			case "d":
				if len(m.journals) > 0 {
					_ = os.Remove(filepath.Join(journalDir(), m.journals[m.cursor].filename))
					m.loadAndSync()
				}
			}
		case stateEdit:
			if msg.String() == "esc" {
				if strings.TrimSpace(m.editor.Value()) != "" {
					m.saveJournal()
				}
				m.loadAndSync()
				m.state = stateNav
				m.editor.Blur()
			}
		}
	}

	m.editor, tiCmd = m.editor.Update(msg)
	m.viewPort, vpCmd = m.viewPort.Update(msg)
	return m, tea.Batch(tiCmd, vpCmd)
}

func (m *model) loadAndSync() {
	m.loadJournals()
	if m.cursor >= len(m.journals) && m.cursor > 0 {
		m.cursor--
	}
	m.syncView()
}

func (m *model) syncView() {
	if len(m.journals) > 0 {
		content := m.journals[m.cursor].content
		lines := strings.Split(content, "\n")
		var styled strings.Builder
		for i, line := range lines {
			if i == 0 {
				styled.WriteString(titleStyle.Render(line) + "\n\n")
			} else {
				styled.WriteString(line + "\n")
			}
		}
		m.viewPort.SetContent(styled.String())
	} else {
		m.viewPort.SetContent("Silence. Press 'n' to begin.")
	}
}

func (m model) View() string {
	if !m.ready { return "" }

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(leaf).Render("ARCHIVE") + "\n\n")
	
	for i, j := range m.journals {
		dateStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#526D82")).Render(" " + j.date)
		if i == m.cursor {
			sb.WriteString(listSelected.Render("● "+j.title) + dateStr + "\n")
		} else {
			sb.WriteString("  " + j.title + dateStr + "\n")
		}
	}

	sb.WriteString("\n\n" + instructionStyle.Render("N      New") + "\n")
	sb.WriteString(instructionStyle.Render("ENT    Edit") + "\n")
	sb.WriteString(instructionStyle.Render("D      Delete") + "\n")
	sb.WriteString(instructionStyle.Render("ESC    Save") + "\n")
	sb.WriteString(instructionStyle.Render("CTRL+C Close") + "\n")
	
	sidebar := sidebarStyle.Width(m.sidebarWidth).Height(m.height).Render(sb.String())

	header := activeTabStyle.Render("ZEN SPACE")
	main := m.viewPort.View()
	if m.state == stateEdit {
		main = m.editor.View()
	}

	content := contentStyle.Width(m.width - m.sidebarWidth).Height(m.height).Render(
		header + "\n\n" + main,
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func (m *model) loadJournals() {
	dir := journalDir()
	files, _ := os.ReadDir(dir)
	var js []journalEntry
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".txt" {
			info, _ := f.Info()
			content, _ := os.ReadFile(filepath.Join(dir, f.Name()))
			text := string(content)
			title := "Untitled"
			lines := strings.Split(text, "\n")
			for _, l := range lines {
				if strings.TrimSpace(l) != "" {
					title = l
					if len(title) > 18 { title = title[:15] + "..." }
					break
				}
			}
			js = append(js, journalEntry{
				filename: f.Name(), 
				title: title, 
				content: text,
				date: info.ModTime().Format("01/02"),
			})
		}
	}
	sort.Slice(js, func(i, j int) bool { return js[i].filename > js[j].filename })
	m.journals = js
}

func (m model) saveJournal() {
	var filename string
	if m.activeFile != "" {
		filename = m.activeFile
	} else {
		filename = time.Now().Format("20060102_150405") + ".txt"
	}
	path := filepath.Join(journalDir(), filename)
	_ = os.WriteFile(path, []byte(m.editor.Value()), 0644)
}

func journalDir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".zen_v5")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func main() {
	m := initialModel()
	m.loadJournals()
	m.syncView()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}