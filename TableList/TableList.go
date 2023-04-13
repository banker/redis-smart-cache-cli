package TableList

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/redis/go-redis/v9"
	"smart-cache-cli/ConfirmationDialog"
	"smart-cache-cli/RedisCommon"
	"smart-cache-cli/RuleTtlView"
	"smart-cache-cli/SortDialog"
	"strings"
)

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

type Model struct {
	parentModel     tea.Model
	table           table.Model
	tables          []RedisCommon.Table
	rdb             *redis.Client
	sortColumn      string
	sortDirection   SortDialog.Direction
	applicationName string
}

func (m Model) Selection() *RedisCommon.Table {
	return &m.tables[m.table.HighlightedRow().Data["RowId"].(int)]
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case tea.KeyCtrlC.String():
			return m, tea.Quit
		case tea.KeyEnter.String():
			return RuleTtlView.New(m.Selection(), m), cmd
		case "b":
			m.parentModel, _ = m.parentModel.Update(ConfirmationDialog.ConfirmationMessage{ConfirmedUpdate: true})
			return m.parentModel, nil

		}
	case RuleTtlView.TableTtlMsg:
		rule := RedisCommon.Rule{
			Ttl:       msg.Ttl,
			TablesAny: []string{m.Selection().Name},
		}
		RedisCommon.CommitNewRules(m.rdb, []RedisCommon.Rule{rule}, m.applicationName)
		ResetModel(&m)
		return m, cmd
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	body := strings.Builder{}
	body.WriteString("Press Enter to update the TTL for a table\n")
	body.WriteString("Press 'b' to go back\n")
	body.WriteString(m.table.View())

	return body.String()
}

func ResetModel(m *Model) {
	tables := RedisCommon.GetTables(m.rdb, m.applicationName)

	rows := make([]table.Row, len(tables))
	for i, t := range tables {
		rows[i] = t.GetAsRow(i)
	}

	m.table = m.table.WithRows(rows)
}

func New(parentModel tea.Model, rdb *redis.Client, applicationName string) Model {
	tables := RedisCommon.GetTables(rdb, applicationName)

	rows := make([]table.Row, len(tables))
	for i, t := range tables {
		rows[i] = t.GetAsRow(i)
	}

	model := Model{
		tables: tables,
		table: table.New(RedisCommon.GetColumnsOfTable("Query Time", SortDialog.Descending)).
			WithRows(rows).
			HeaderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)).
			Focused(true).
			Border(customBorder).
			WithPageSize(5).
			SortByDesc("Query Time").WithTargetWidth(200),
		parentModel:     parentModel,
		rdb:             rdb,
		applicationName: applicationName,
		sortColumn:      "Query Time",
		sortDirection:   SortDialog.Descending,
	}

	return model
}
