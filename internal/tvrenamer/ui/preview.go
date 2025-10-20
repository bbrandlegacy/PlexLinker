package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"plexlinker/internal/tvrenamer/filesystem"
	"plexlinker/internal/tvrenamer/patterns"
)

type PreviewItem struct {
	CurrentName string
	NewName     string
	Path        string
	HasConflict bool
	Error       string
}

type PreviewUI struct {
	screen       *tcell.Screen
	items        []PreviewItem
	scrollOffset int
	conflicts    []string
	statusMessage string
}

func NewPreviewUI(screen *tcell.Screen, selectedFiles map[string]*filesystem.FileItem) *PreviewUI {
	plexNamer := patterns.NewPlexNamer(nil)

	var items []PreviewItem
	var conflicts []string
	nameCount := make(map[string]int)

	// Generate preview items and check for conflicts
	for path, item := range selectedFiles {
		previewItem := PreviewItem{
			CurrentName: item.Name,
			Path:        path,
		}

		if item.TVInfo != nil {
			previewItem.NewName = plexNamer.GeneratePlexName(item.TVInfo)
			nameCount[previewItem.NewName]++
		} else {
			previewItem.Error = "Could not parse TV show information"
			previewItem.NewName = item.Name // Keep original name
		}

		items = append(items, previewItem)
	}

	// Mark conflicts
	for i := range items {
		if nameCount[items[i].NewName] > 1 {
			items[i].HasConflict = true
		}
	}

	// Generate conflict list
	for name, count := range nameCount {
		if count > 1 {
			conflicts = append(conflicts, fmt.Sprintf("%s (%d files)", name, count))
		}
	}

	// Sort items by current name
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].CurrentName) < strings.ToLower(items[j].CurrentName)
	})

	return &PreviewUI{
		screen:    screen,
		items:     items,
		conflicts: conflicts,
		statusMessage: fmt.Sprintf("Preview for %d files", len(items)),
	}
}

func (ui *PreviewUI) HandleKeyEvent(ev *tcell.EventKey) bool {
	_, h := (*ui.screen).Size()
	maxVisibleItems := h - 8 // Account for header, conflicts, and footer

	switch ev.Key() {
	case tcell.KeyUp:
		ui.scrollUp()
	case tcell.KeyDown:
		ui.scrollDown(maxVisibleItems)
	case tcell.KeyPgUp:
		ui.scrollPageUp(maxVisibleItems)
	case tcell.KeyPgDn:
		ui.scrollPageDown(maxVisibleItems)
	case tcell.KeyHome:
		ui.scrollOffset = 0
	case tcell.KeyEnd:
		ui.scrollToEnd(maxVisibleItems)
	case tcell.KeyEscape:
		return true
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'k':
			ui.scrollUp()
		case 'j':
			ui.scrollDown(maxVisibleItems)
		case 'q':
			return true
		case 'g':
			ui.scrollOffset = 0
		case 'G':
			ui.scrollToEnd(maxVisibleItems)
		}
	}

	return false
}

func (ui *PreviewUI) scrollUp() {
	if ui.scrollOffset > 0 {
		ui.scrollOffset--
	}
}

func (ui *PreviewUI) scrollDown(maxVisible int) {
	maxScroll := len(ui.items) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if ui.scrollOffset < maxScroll {
		ui.scrollOffset++
	}
}

func (ui *PreviewUI) scrollPageUp(maxVisible int) {
	ui.scrollOffset -= maxVisible / 2
	if ui.scrollOffset < 0 {
		ui.scrollOffset = 0
	}
}

func (ui *PreviewUI) scrollPageDown(maxVisible int) {
	maxScroll := len(ui.items) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	ui.scrollOffset += maxVisible / 2
	if ui.scrollOffset > maxScroll {
		ui.scrollOffset = maxScroll
	}
}

func (ui *PreviewUI) scrollToEnd(maxVisible int) {
	maxScroll := len(ui.items) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	ui.scrollOffset = maxScroll
}

func (ui *PreviewUI) Draw() {
	screen := *ui.screen
	w, h := screen.Size()
	screen.Clear()

	ui.drawHeader(w)
	ui.drawConflicts(w, h)
	ui.drawPreviewList(w, h)
	ui.drawStatusBar(w, h)

	screen.Show()
}

func (ui *PreviewUI) drawHeader(w int) {
	title := "PlexLinker - Rename Preview"
	ui.drawText(0, 0, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite), title)

	subtitle := fmt.Sprintf("Previewing %d files for rename", len(ui.items))
	ui.drawText(0, 1, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), subtitle)
}

func (ui *PreviewUI) drawConflicts(w, h int) {
	startY := 3
	if len(ui.conflicts) > 0 {
		conflictTitle := "⚠️  CONFLICTS DETECTED:"
		ui.drawText(0, startY, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorRed), conflictTitle)

		maxConflicts := 3 // Show max 3 conflicts
		for i, conflict := range ui.conflicts {
			if i >= maxConflicts {
				more := fmt.Sprintf("... and %d more conflicts", len(ui.conflicts)-maxConflicts)
				ui.drawText(2, startY+1+i, tcell.StyleDefault.Foreground(tcell.ColorRed), more)
				break
			}
			ui.drawText(2, startY+1+i, tcell.StyleDefault.Foreground(tcell.ColorRed), conflict)
		}
	} else {
		noConflicts := "✅ No naming conflicts detected"
		ui.drawText(0, startY, tcell.StyleDefault.Foreground(tcell.ColorGreen), noConflicts)
	}
}

func (ui *PreviewUI) drawPreviewList(w, h int) {
	listStartY := 6
	if len(ui.conflicts) > 0 {
		listStartY = 6 + min(len(ui.conflicts), 3)
	}

	maxItems := h - listStartY - 3 // Leave space for status bar

	headerLine := "Current Name → New Name"
	ui.drawText(1, listStartY, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite), headerLine)

	separator := strings.Repeat("─", w-2)
	ui.drawText(1, listStartY+1, tcell.StyleDefault.Foreground(tcell.ColorWhite), separator)

	for i := 0; i < maxItems && i+ui.scrollOffset < len(ui.items); i++ {
		itemIndex := i + ui.scrollOffset
		item := ui.items[itemIndex]
		y := listStartY + 2 + i

		var style tcell.Style
		var indicator string

		if item.Error != "" {
			style = tcell.StyleDefault.Foreground(tcell.ColorRed)
			indicator = "❌"
		} else if item.HasConflict {
			style = tcell.StyleDefault.Foreground(tcell.ColorYellow)
			indicator = "⚠️ "
		} else if item.CurrentName == item.NewName {
			style = tcell.StyleDefault.Foreground(tcell.ColorLightGray)
			indicator = "➡️ "
		} else {
			style = tcell.StyleDefault.Foreground(tcell.ColorGreen)
			indicator = "✅"
		}

		// Format: indicator currentname → newname
		text := fmt.Sprintf("%s %s → %s", indicator, item.CurrentName, item.NewName)
		if len(text) > w-2 {
			text = text[:w-5] + "..."
		}

		ui.drawText(1, y, style, text)
	}

	// Draw scroll indicator
	if len(ui.items) > maxItems {
		scrollInfo := fmt.Sprintf("(%d-%d of %d)",
			ui.scrollOffset+1,
			min(ui.scrollOffset+maxItems, len(ui.items)),
			len(ui.items))

		ui.drawText(w-len(scrollInfo), listStartY, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), scrollInfo)
	}
}

func (ui *PreviewUI) drawStatusBar(w, h int) {
	statusY := h - 2
	commandsY := h - 1

	statusText := ui.statusMessage
	if len(ui.conflicts) > 0 {
		statusText = fmt.Sprintf("⚠️  %d conflicts detected - review before proceeding", len(ui.conflicts))
	}

	if len(statusText) > w {
		statusText = statusText[:w-3] + "..."
	}
	ui.drawText(0, statusY, tcell.StyleDefault.Foreground(tcell.ColorYellow), statusText)

	commands := "↑↓/jk:Scroll  PgUp/PgDn:Page  Home/g:Top  End/G:Bottom  Esc/q:Exit"
	if len(commands) > w {
		commands = "↑↓:Scroll  PgUp/Dn:Page  Home:Top  End:Bottom  Esc:Exit"
	}

	ui.drawText(0, commandsY, tcell.StyleDefault.Foreground(tcell.ColorWhite), commands)
}

func (ui *PreviewUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *ui.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}