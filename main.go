package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
)

var (
	Program     *tea.Program
	senderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	botStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	footerStyle = lipgloss.NewStyle().
			Height(1).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("8")).
			Faint(true)
)

type KeyMap struct {
	Up   key.Binding
	Down key.Binding
}

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Submit key.Binding
	Clear  key.Binding
	Reload key.Binding
	Quit   key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("ctrl+p", "pgup"),    // actual keybindings
		key.WithHelp("ctrl+p", "move up"), // corresponding help text
	),
	Down: key.NewBinding(
		key.WithKeys("ctrl+n", "pgdown"),
		key.WithHelp("ctrl+n", "move down"),
	),
	Submit: key.NewBinding(
		key.WithKeys("ctrl+j"),
		key.WithHelp("ctrl+j", "send message"),
	),
	Clear: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("ctrl+l", "clear chat history"),
	),
	Reload: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "(debug) reload copilot token"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
}

var viewportKeyMap = viewport.KeyMap{
	Up:   keys.Up,
	Down: keys.Down,
}

type HistoryMessage struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Submit, k.Up, k.Down, k.Quit, k.Clear, k.Reload}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Submit, k.Quit}, // first column
		{k.Clear, k.Reload},              // second column
	}
}

type model struct {
	messages       []HistoryMessage
	history        []HistoryMessage
	textarea       textarea.Model
	viewport       viewport.Model
	copilotRequest CopilotRequest
	answering      bool
	keys           keyMap
	help           help.Model
	ready          bool
	width          int
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Write your query..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 1000

	ta.SetHeight(4)

	ta.ShowLineNumbers = true

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	initialModel := model{
		messages: []HistoryMessage{
			createBotHistoryEntry("How can I assist you today?"),
		},
		textarea:       ta,
		copilotRequest: generateCopilotRequest(),
		answering:      false,
		keys:           keys,
		help:           help.New(),
	}

	initialModel.history = append(initialModel.history, createSystemHistoryEntry(SYSTEM_PROMPT))

	return initialModel
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{textarea.Blink}

	return tea.Batch(cmds...)
}

func (m model) headerView() string {
	return lipgloss.JoinHorizontal(lipgloss.Center, m.viewport.View())
}

func (m model) View() string {
	var views []string

	views = append(views, m.viewport.View())
	views = append(views, m.footerView())
	views = append(views, m.helpView())

	return lipgloss.JoinVertical(lipgloss.Top, views...)
}

func (m model) helpView() string {
	help := m.help.View(m.keys)

	return help
}

func createHistoryEntry(msg string) HistoryMessage {
	return HistoryMessage{
		Content: msg,
		Role:    "user",
	}
}

func createBotHistoryEntry(msg string) HistoryMessage {
	return HistoryMessage{
		Content: msg,
		Role:    "assistant",
	}
}

func createSystemHistoryEntry(msg string) HistoryMessage {
	return HistoryMessage{
		Content: msg,
		Role:    "system",
	}
}

type LoadingMsg struct{}
type ResponseMsg struct{}
type AnswerMsg struct {
	content string
	done    bool
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case LoadingMsg:
		message := m.textarea.Value()

		m.history = append(m.history, createHistoryEntry(message))

		m.messages = append(m.messages, createHistoryEntry(message))

		m.textarea.SetValue("Loading...")

		cmds = append(cmds, func() tea.Msg { return ResponseMsg{} })

	case AnswerMsg:
		m.messages[len(m.messages)-1].Content = msg.content

		str := renderMessages(m.messages, m.width)

		m.viewport.SetContent(str)

		m.viewport.GotoBottom()

		if msg.done {
			m.textarea.Reset()

			break
		}

	case ResponseMsg:
		if m.answering {
			break
		}

		m.messages = append(m.messages, createBotHistoryEntry("Thinking..."))

		cmds = append(cmds, func() tea.Msg {
			getResponse(&m, func(reply string, done bool) {

				if done {
					m.answering = false

					return
				}

				Program.Send(AnswerMsg{content: reply, done: true})
			})

			m.answering = true

			return nil
		})

	case tea.WindowSizeMsg:
		footerHeight := lipgloss.Height(m.footerView())
		viewportHeight := msg.Height - footerHeight - lipgloss.Height(m.helpView())

		m.width = msg.Width

		m.textarea.SetWidth(msg.Width)

		if !m.ready {
			m.ready = true

			m.viewport = viewport.New(msg.Width, viewportHeight)
			m.viewport.KeyMap = viewportKeyMap
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = viewportHeight
		}

		str := renderMessages(m.messages, msg.Width)

		m.viewport.SetContent(str)

	case tea.KeyMsg:
		switch {

		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Clear):
			m.history = m.history[:1]
			m.messages = m.messages[:1]

			m.viewport.GotoBottom()

			str := renderMessages(m.messages, m.width)
			m.viewport.SetContent(str)

		case key.Matches(msg, m.keys.Reload):
			m.copilotRequest = generateCopilotRequest()

		case key.Matches(msg, m.keys.Submit):
			if m.textarea.Value() != "" {
				cmds = append(cmds, func() tea.Msg { return LoadingMsg{} })
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) footerView() string {
	line := strings.Repeat("─", m.viewport.Width)

	return lipgloss.JoinVertical(lipgloss.Bottom, line, m.textarea.View())
}

func main() {
	debug := flag.Bool("d", false, "Enable debug mode")

	flag.Parse()

	if *debug {
		file, err := os.OpenFile("gopilot.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

		if err != nil {
			log.Fatal(err)
		}

		defer file.Close()

		log.SetOutput(file)
	} else {
		log.SetOutput(io.Discard)
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())

	Program = p

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running the program: %v", err)

		os.Exit(1)
	}
}

func renderText(str string, width int) string {
	str = wrap.String(str, width)

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)

	response, _ := renderer.Render(str)

	return response
}

func renderBotText(str string, width int) string {
	return botStyle.Render("GitHub Copilot: ") + renderText(str, width)
}

func renderUserText(str string, width int) string {
	return senderStyle.Render("You: ") + renderText(str, width)
}

func renderMessages(messages []HistoryMessage, width int) string {
	wrappedStrings := make([]string, len(messages))

	for i, message := range messages {
		if message.Role == "assistant" {
			wrappedStrings[i] = renderBotText(message.Content, width)
			continue
		}

		if message.Role == "user" {
			wrappedStrings[i] = renderUserText(message.Content, width)
			continue
		}

		wrappedStrings[i] = renderText(message.Content, width)
	}

	return strings.Join(wrappedStrings, "")
}
