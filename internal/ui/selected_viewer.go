package ui

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"plexlinker/internal/browser"
)

type SelectedViewerUI struct {
	screen        *tcell.Screen
	selectedFiles map[string]*browser.DirectoryItem
	sortedFiles   []string
	cursorIndex   int
	scrollOffset  int
	needsRefresh  bool // Flag to indicate list needs to be refreshed
}

func NewSelectedViewerUI(screen *tcell.Screen, selectedFiles map[string]*browser.DirectoryItem) *SelectedViewerUI {
	// Sort the files for consistent display
	sortedFiles := make([]string, 0, len(selectedFiles))
	for url := range selectedFiles {
		sortedFiles = append(sortedFiles, url)
	}
	sort.Strings(sortedFiles)

	return &SelectedViewerUI{
		screen:        screen,
		selectedFiles: selectedFiles,
		sortedFiles:   sortedFiles,
		cursorIndex:   0,
		scrollOffset:  0,
	}
}

func (ui *SelectedViewerUI) HandleKeyEvent(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		ui.moveCursorUp()
	case tcell.KeyDown:
		ui.moveCursorDown()
	case tcell.KeyEnter:
		ui.removeCurrentFile()
	case tcell.KeyEscape, tcell.KeyBackspace:
		return true // Exit viewer
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'k':
			ui.moveCursorUp()
		case 'j':
			ui.moveCursorDown()
		case 'q':
			return true // Exit viewer
		case '3':
			ui.removeAllDisplayed()
		}
	}
	return false
}

func (ui *SelectedViewerUI) removeCurrentFile() {
	if len(ui.sortedFiles) == 0 {
		return
	}

	if ui.cursorIndex >= len(ui.sortedFiles) {
		return
	}

	// Remove the file at cursor
	url := ui.sortedFiles[ui.cursorIndex]
	delete(ui.selectedFiles, url)

	// Rebuild sorted list
	ui.sortedFiles = make([]string, 0, len(ui.selectedFiles))
	for url := range ui.selectedFiles {
		ui.sortedFiles = append(ui.sortedFiles, url)
	}
	sort.Strings(ui.sortedFiles)

	// Adjust cursor if needed
	if ui.cursorIndex >= len(ui.sortedFiles) && ui.cursorIndex > 0 {
		ui.cursorIndex = len(ui.sortedFiles) - 1
	}
	if len(ui.sortedFiles) == 0 {
		ui.cursorIndex = 0
	}

	ui.adjustScrollForCursor()
	ui.needsRefresh = true
}

func (ui *SelectedViewerUI) removeAllDisplayed() {
	// Remove all files from selection
	for _, url := range ui.sortedFiles {
		delete(ui.selectedFiles, url)
	}

	// Clear the list
	ui.sortedFiles = []string{}
	ui.cursorIndex = 0
	ui.scrollOffset = 0
	ui.needsRefresh = true
}

func (ui *SelectedViewerUI) moveCursorUp() {
	if ui.cursorIndex > 0 {
		ui.cursorIndex--
		ui.adjustScrollForCursor()
	}
}

func (ui *SelectedViewerUI) moveCursorDown() {
	if ui.cursorIndex < len(ui.sortedFiles)-1 {
		ui.cursorIndex++
		ui.adjustScrollForCursor()
	}
}

func (ui *SelectedViewerUI) adjustScrollForCursor() {
	_, h := (*ui.screen).Size()
	maxVisibleItems := h - 5 // Account for header and status bar

	// If cursor is above visible area, scroll up
	if ui.cursorIndex < ui.scrollOffset {
		ui.scrollOffset = ui.cursorIndex
	}

	// If cursor is below visible area, scroll down
	if ui.cursorIndex >= ui.scrollOffset+maxVisibleItems {
		ui.scrollOffset = ui.cursorIndex - maxVisibleItems + 1
	}

	// Ensure scroll offset doesn't go negative
	if ui.scrollOffset < 0 {
		ui.scrollOffset = 0
	}
}

func (ui *SelectedViewerUI) Draw() {
	screen := *ui.screen
	w, h := screen.Size()
	screen.Clear()

	ui.drawHeader(w)
	ui.drawFileList(w, h)
	ui.drawStatusBar(w, h)

	screen.Show()
}

func (ui *SelectedViewerUI) drawHeader(w int) {
	title := "Selected Files"
	ui.drawText(0, 0, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite), title)

	countText := fmt.Sprintf("Total: %d files", len(ui.selectedFiles))
	ui.drawText(w-len(countText), 0, tcell.StyleDefault.Foreground(tcell.ColorGreen), countText)
}

func (ui *SelectedViewerUI) drawFileList(w, h int) {
	startY := 2
	maxItems := h - 4

	// Draw visible items based on scroll offset
	for i := 0; i < maxItems; i++ {
		itemIndex := i + ui.scrollOffset
		if itemIndex >= len(ui.sortedFiles) {
			break
		}

		url := ui.sortedFiles[itemIndex]
		item := ui.selectedFiles[url]
		y := startY + i
		style := tcell.StyleDefault

		if itemIndex == ui.cursorIndex {
			style = style.Background(tcell.ColorNavy)
		}

		icon := item.GetFileIcon()
		style = style.Foreground(tcell.ColorWhite)

		text := fmt.Sprintf("%s %s", icon, item.Name)
		if len(text) > w-2 {
			text = text[:w-5] + "..."
		}

		ui.drawText(1, y, style, text)
	}

	// Draw scroll indicator if needed
	if len(ui.sortedFiles) > maxItems {
		scrollInfo := fmt.Sprintf("(%d-%d of %d)",
			ui.scrollOffset+1,
			ui.scrollOffset+maxItems,
			len(ui.sortedFiles))
		if ui.scrollOffset+maxItems > len(ui.sortedFiles) {
			scrollInfo = fmt.Sprintf("(%d-%d of %d)",
				ui.scrollOffset+1,
				len(ui.sortedFiles),
				len(ui.sortedFiles))
		}

		ui.drawText(w-len(scrollInfo), startY-1, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), scrollInfo)
	}
}

func (ui *SelectedViewerUI) drawStatusBar(w, h int) {
	statusY := h - 2
	commandsY := h - 1

	statusText := "Browse selected files (press Esc or q to return)"
	ui.drawText(0, statusY, tcell.StyleDefault.Foreground(tcell.ColorYellow), statusText)

	commands := "Enter:Remove  3:RemoveAll  ↑↓/jk:Navigate  Esc/q:Exit"
	if len(commands) > w {
		commands = "Enter:Remove  3:Clear  Esc:Exit"
	}
	ui.drawText(0, commandsY, tcell.StyleDefault.Foreground(tcell.ColorWhite), commands)
}

func (ui *SelectedViewerUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *ui.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}