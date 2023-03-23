package queryList

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/redis/go-redis/v9"
	"rsccli/ConfirmationDialog"
	"rsccli/RedisCommon"
	"rsccli/queryTtlView"
	"strings"
)

const listHeight = 14

type Model struct {
	parentModel  tea.Model
	table        table.Model
	Queries      []*RedisCommon.Query
	pendingRules map[string]RedisCommon.Rule
	Selection    int
}

var (
	customBorder = table.Border{
		Top:    "─",
		Left:   "│",
		Right:  "│",
		Bottom: "─",

		TopRight:    "╮",
		TopLeft:     "╭",
		BottomRight: "╯",
		BottomLeft:  "╰",

		TopJunction:    "╥",
		LeftJunction:   "├",
		RightJunction:  "┤",
		BottomJunction: "╨",
		InnerJunction:  "╫",

		InnerDivider: "║",
	}
)

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) updateFooter() table.Model {

	footerText := fmt.Sprintf(
		"Pg. %d/%d - Pending Updates: %d",
		m.table.CurrentPage(),
		m.table.MaxPages(),
		len(m.pendingRules),
	)

	return m.table.WithStaticFooter(footerText)
}

func (m Model) UpdateCurrentTtl(ttl string) {
	m.table.HighlightedRow().Data["Pending Rule"] = ttl
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.table, cmd = m.table.Update(msg)

	m.table = m.updateFooter()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		switch s {
		case tea.KeyCtrlC.String(), tea.KeyEsc.String(), "q":
			return m.parentModel, tea.Quit
		case tea.KeyTab.String(), tea.KeySpace.String(), tea.KeyEnter.String():
			m.Selection = m.table.HighlightedRow().Data["RowId"].(int)
			//m.EditMode = !m.EditMode
			return queryTtlView.New(m.Queries[m.Selection], m), cmd
		case "i":
			m.table = m.table.WithHeaderVisibility(!m.table.GetHeaderVisibility())
		case "c":
			return ConfirmationDialog.New(m, m.pendingRules), cmd
		}
	case queryTtlView.SetPendingTtlMsg:
		m.table.HighlightedRow().Data["Pending Rule"] = msg.Ttl
		r, ok := m.pendingRules[msg.Ttl]
		if ok {
			r.QueryIds = append(r.QueryIds, m.Queries[m.Selection].Id)
		} else {
			m.pendingRules[msg.Ttl] = RedisCommon.Rule{
				Ttl:      msg.Ttl,
				QueryIds: []string{m.Queries[m.Selection].Id},
			}
		}
	}

	return m, cmd
}

func (m Model) View() string {
	body := strings.Builder{}
	m.table = m.updateFooter()

	body.WriteString("Press left/right or page up/down to move pages\n")
	body.WriteString("Press 'i' to toggle the header visibility\n")
	body.WriteString("Press space/enter to create a pending rule\n")
	body.WriteString("Press 'c' to commit selected rules\n")
	body.WriteString("Press q or ctrl+c to quit\n")

	body.WriteString(m.table.View())

	body.WriteString("\n")

	return body.String()
}

func InitialModel(pm tea.Model) Model {
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", "localhost", "6379"),
		DB:   0,
	})
	queries, err := RedisCommon.GetQueries(rdb)

	if err != nil {
		println(err)
	}

	rows := make([]table.Row, len(queries))
	for i, q := range queries {
		rows[i] = q.GetAsRow(i)
	}
	model := Model{
		table: table.New(RedisCommon.GetColumnsOfQuery()).
			WithRows(rows).
			HeaderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)).
			Focused(true).
			Border(customBorder).
			WithPageSize(5).
			SortByDesc("Mean Query Time"),
		Queries:      queries,
		parentModel:  pm,
		pendingRules: make(map[string]RedisCommon.Rule),
	}
	model.table = model.updateFooter()

	return model
}
