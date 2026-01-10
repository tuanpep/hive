package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type TaskDelegate struct{}

func (d TaskDelegate) Height() int                               { return 2 }
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

	titleStr := fmt.Sprintf("[%s] %s", shortID, it.Title)
	if len(titleStr) > 25 {
		titleStr = titleStr[:25] + "..."
	}

	logStr := it.LastLog
	if logStr == "" {
		logStr = "Waiting..."
	}
	if len(logStr) > 30 {
		logStr = logStr[:27] + "..."
	}

	if index == m.Index() {
		fmt.Fprint(w, StyleTaskSelected.Render(fmt.Sprintf("> %s", titleStr))+"\n")
		// Selected item secondary line (maybe distinct color?)
		fmt.Fprint(w, StyleDimmed.Render(fmt.Sprintf("    %s", logStr)))
	} else {
		fmt.Fprint(w, StyleTaskDimmed.Render(fmt.Sprintf("  %s", titleStr))+"\n")
		fmt.Fprint(w, StyleDimmed.Render(fmt.Sprintf("    %s", logStr)))
	}
}
