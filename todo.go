package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type item struct {
	ID          int
	Description string
	Done        bool
}

func (i item) FilterValue() string { return i.Description }

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).Background(lipgloss.Color("10"))
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type itemDelegate struct {
}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, l *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(item)
	if !ok {
		return
	}

	var str string
	if item.Done {
		str = fmt.Sprintf("[x] %s", item.Description)
	} else {
		str = fmt.Sprintf("[ ] %s", item.Description)
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))

}

type model struct {
	list       list.Model
	input      textinput.Model
	inputStyle lipgloss.Style
	OnNew      func(item)
	OnUpdate   func(item)
	OnDelete   func(item)
	Refresh    func([]item)
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case tea.KeyMsg:
		if m.input.Focused() {
			if msg.Type == tea.KeyEsc {
				m.input.Blur()
			} else if msg.Type == tea.KeyEnter {
				m.input.Blur()
				itm := item{
					Description: m.input.Value(),
					Done:        false,
					ID:          len(m.list.Items()),
				}
				if m.OnNew != nil {
					m.OnNew(itm)
				}
				cmd = m.list.InsertItem(len(m.list.Items()), itm)
				cmds = append(cmds, cmd)
				m.input.SetValue("")
			}
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			switch keypress := msg.String(); keypress {
			case "q", "ctrl+c":
				if m.list.FilterState() != list.Filtering && !m.input.Focused() {
					return m, tea.Quit
				}
			case " ":
				i := m.list.SelectedItem().(item)
				i.Done = !i.Done
				m.OnUpdate(i)
				m.list.SetItem(m.list.Index(), i)
			case "n":
				m.input.Focus()
				cmds = append(cmds, textinput.Blink)
			case "d":
				m.OnDelete(m.list.SelectedItem().(item))
				m.list.RemoveItem(m.list.Index())

			}
		}
	}

	if !m.input.Focused() {
		m.list, cmd = m.list.Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	list := "\n" + m.list.View()

	if m.input.Focused() {
		return lipgloss.Place(76, 13, lipgloss.Center, lipgloss.Center,
			m.inputStyle.Render(m.input.View()),
		)
	}
	return list
}

func NewModel(items []item) model {

	listItems := []list.Item{}
	for _, item := range items {
		listItems = append(listItems, item)
	}

	l := list.New(listItems, itemDelegate{}, 20, 12)
	l.Title = "Todos"
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
			key.NewBinding(key.WithKeys(" "), key.WithHelp(" ", "done")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
			key.NewBinding(key.WithKeys(" "), key.WithHelp(" ", "done")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		}
	}

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)

	input := textinput.New()
	input.Placeholder = "New todo"
	input.Prompt = ""
	input.Width = 40

	return model{list: l, input: input, inputStyle: inputStyle}
}
