package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

var dataDir = "~/.warlog"

type model struct {
	list       list.Model
	textarea   textarea.Model
	state      appState
	selected   string
	files      []string
	fileViewer *glamour.TermRenderer
}

type appState int

const (
	listView appState = iota
	editView
	viewerView
)

func main() {
	// if the environment variable WAR_LOG_DATA is set, use it as the data directory else use the constant dataDir
	if dataDirEnv := os.Getenv("WAR_LOG_DATA"); dataDirEnv != "" {
		dataDir = dataDirEnv
	}
	// if the directory does not exist, create it
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.Mkdir(dataDir, 0755); err != nil {
			log.Fatalf("failed to create data directory: %v", err)
		}
	}

	files := loadFiles()

	// Set up the file list
	items := make([]list.Item, len(files))
	for i, file := range files {
		items[i] = listItem{title: filepath.Base(file), filePath: file}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 10)
	l.Title = "War Log Files"
	l.AdditionalShortHelpKeys = extraKeys
	l.AdditionalFullHelpKeys = extraKeys

	ta := textarea.New()
	ta.Placeholder = "Enter markdown text here..."
	ta.Focus()

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
	)
	if err != nil {
		log.Fatalf("failed to create renderer: %v", err)
	}

	m := model{
		list:       l,
		textarea:   ta,
		state:      listView,
		files:      files,
		fileViewer: renderer,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}

func extraKeys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "New")),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+q"))):
			return m, tea.Quit

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+n"))):
			if m.state == listView {
				m.textarea.Reset()
				m.textarea.SetValue(fmt.Sprintf("%s\n\n", time.Now().Format("2006-01-02 15:04:05")))
				m.state = editView
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+s"))):
			if m.state == editView {
				m.saveFile(m.textarea.Value())
				m.state = listView
				m.files = loadFiles()
				m.refreshList()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+b"))):
			if m.state == editView || m.state == viewerView {
				m.state = listView
			}
		case msg.String() == "enter":
			if m.state == listView {
				m.selected = m.files[m.list.Index()]
				m.state = viewerView
			}
		}

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-1)
		m.textarea.SetWidth(msg.Width - 2)
	}

	switch m.state {
	case listView:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case editView:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	case viewerView:
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	switch m.state {
	case listView:
		return m.list.View()
	case editView:
		return fmt.Sprintf("%s\n%s\n%s", mainHeader("New Entry"), m.textarea.View(), commandBar("ctrl+s: Save | ctrl+b: Cancel"))
	case viewerView:
		content, _ := os.ReadFile(m.selected)
		out, _ := m.fileViewer.Render(string(content))
		return fmt.Sprintf("%s\n%s\n%s", mainHeader(m.selected), out, commandBar("ctrl+b: Go back | ctrl+q: Quit"))
	default:
		return ""
	}
}

func (m *model) saveFile(content string) {
	fileName := fmt.Sprintf("%s-log.md", time.Now().Format("2006-01-02-15-04"))
	filePath := filepath.Join(dataDir, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		log.Fatalf("failed to save file: %v", err)
	}
}

func (m *model) refreshList() {
	items := make([]list.Item, len(m.files))
	for i, file := range m.files {
		items[i] = listItem{title: filepath.Base(file), filePath: file}
	}
	m.list.SetItems(items)
}

type listItem struct {
	title    string
	filePath string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.filePath }
func (i listItem) FilterValue() string { return i.title }

func loadFiles() []string {
	var files []string

	_ = filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// only add files with extension .md to the list
		if !info.IsDir() && filepath.Ext(path) == ".md" {
			files = append(files, path)
		}
		return nil
	})

	sort.Slice(files, func(i, j int) bool {
		iInfo, _ := os.Stat(files[i])
		jInfo, _ := os.Stat(files[j])

		if iInfo.ModTime() == jInfo.ModTime() {
			return files[i] < files[j]
		}
		return iInfo.ModTime().Before(jInfo.ModTime())
	})

	return files
}

func mainHeader(message string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#05d9e8")).
		Foreground(lipgloss.Color("#d1f7ff")).
		Background(lipgloss.Color("#01012b")).
		PaddingBottom(0).
		PaddingLeft(4).
		Width(80).
		Render(message)
}

func commandBar(options string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#05d9e8")).
		Foreground(lipgloss.Color("#d1f7ff")).
		Background(lipgloss.Color("#01012b")).
		PaddingLeft(4).
		Width(80).
		Render(options)
}

func applyLipglossBorder(content string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#05d9e8")).
		Padding(0, 0, 0, 0).
		Render(content)
}
