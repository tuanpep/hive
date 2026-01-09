package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type TaskDelegate struct{}

func (d TaskDelegate) Height() int                               { return 1 }
func (d TaskDelegate) Spacing() int                              { return 0 }
func (d TaskDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d TaskDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(TaskItem)
	if !ok {
		return
	}

	shortID := it.ID
	if len(it.ID) > 8 {
		shortID = it.ID[len(it.ID)-8:]
	}

	str := fmt.Sprintf("[%s] %s", shortID, it.Title)

	if index == m.Index() {
		fmt.Fprint(w, StyleTaskSelected.Render(fmt.Sprintf("> %s", str)))
	} else {
		fmt.Fprint(w, StyleTaskDimmed.Render(fmt.Sprintf("  %s", str)))
	}
}
