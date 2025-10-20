package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// DirectoryBrowserUI provides a filesystem browser with arrow key navigation
// Up/Down: Navigate list
// Left: Go to parent directory
// Right: Enter selected directory
// Enter: Select current directory
type DirectoryBrowserUI struct {
	screen         *tcell.Screen
	currentPath    string
	entries        []os.DirEntry
	selectedIndex  int
	scrollOffset   int
	statusMessage  string
	title          string
}

func NewDirectoryBrowserUI(screen *tcell.Screen, initialPath, title string) *DirectoryBrowserUI {
	// Start with a reasonable default if path is empty or "auto"
	if initialPath == "" || initialPath == "auto" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			initialPath = "/"
		} else {
			initialPath = homeDir
		}
	}

	// Ensure absolute path
	absPath, err := filepath.Abs(initialPath)
	if err != nil {
		absPath = initialPath
	}

	ui := &DirectoryBrowserUI{
		screen:  screen,
		title:   title,
	}

	ui.loadDirectory(absPath)
	return ui
}

func (ui *DirectoryBrowserUI) loadDirectory(path string) {
	// Read directory contents
	entries, err := os.ReadDir(path)
	if err != nil {
		ui.statusMessage = fmt.Sprintf("Error reading directory: %v", err)
		return
	}

	// Filter to only show directories
	dirs := make([]os.DirEntry, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			// Skip hidden directories (starting with .)
			if !strings.HasPrefix(entry.Name(), ".") {
				dirs = append(dirs, entry)
			}
		}
	}

	// Sort alphabetically
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Name() < dirs[j].Name()
	})

	ui.currentPath = path
	ui.entries = dirs
	ui.selectedIndex = 0
	ui.scrollOffset = 0
	ui.statusMessage = ""
}

func (ui *DirectoryBrowserUI) HandleKeyEvent(ev *tcell.EventKey) *string {
	_, h := (*ui.screen).Size()
	viewableLines := h - 6 // Account for header/footer

	switch ev.Key() {
	case tcell.KeyEscape:
		// Return nil to cancel
		return nil

	case tcell.KeyEnter:
		// Return current path
		result := ui.currentPath
		return &result

	case tcell.KeyUp:
		if ui.selectedIndex > 0 {
			ui.selectedIndex--
			if ui.selectedIndex < ui.scrollOffset {
				ui.scrollOffset = ui.selectedIndex
			}
		}

	case tcell.KeyDown:
		if ui.selectedIndex < len(ui.entries)-1 {
			ui.selectedIndex++
			if ui.selectedIndex-ui.scrollOffset >= viewableLines {
				ui.scrollOffset++
			}
		}

	case tcell.KeyLeft:
		// Go to parent directory
		ui.goToParent()

	case tcell.KeyRight:
		// Enter selected directory
		ui.enterSelectedDirectory()

	case tcell.KeyRune:
		switch ev.Rune() {
		case 'k':
			if ui.selectedIndex > 0 {
				ui.selectedIndex--
				if ui.selectedIndex < ui.scrollOffset {
					ui.scrollOffset = ui.selectedIndex
				}
			}
		case 'j':
			if ui.selectedIndex < len(ui.entries)-1 {
				ui.selectedIndex++
				if ui.selectedIndex-ui.scrollOffset >= viewableLines {
					ui.scrollOffset++
				}
			}
		case 'h':
			ui.goToParent()
		case 'l':
			ui.enterSelectedDirectory()
		case 'q':
			return nil
		}
	}

	return nil // Continue browsing
}

func (ui *DirectoryBrowserUI) goToParent() {
	parentPath := filepath.Dir(ui.currentPath)
	// Don't go above root
	if parentPath != ui.currentPath {
		ui.loadDirectory(parentPath)
	} else {
		ui.statusMessage = "Already at root directory"
	}
}

func (ui *DirectoryBrowserUI) enterSelectedDirectory() {
	if ui.selectedIndex >= 0 && ui.selectedIndex < len(ui.entries) {
		selectedEntry := ui.entries[ui.selectedIndex]
		newPath := filepath.Join(ui.currentPath, selectedEntry.Name())
		ui.loadDirectory(newPath)
	}
}

func (ui *DirectoryBrowserUI) Draw() {
	screen := *ui.screen
	screen.Clear()
	w, h := screen.Size()

	// Header
	headerStyle := tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorWhite).Bold(true)
	headerText := fmt.Sprintf(" %s ", ui.title)
	ui.drawTextCentered(0, w, headerStyle, headerText)

	// Current path
	pathY := 2
	pathStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
	pathText := fmt.Sprintf("Path: %s", ui.currentPath)
	ui.drawText(2, pathY, pathStyle, pathText)

	// Directory listing
	listStartY := 4
	viewableLines := h - 6

	if len(ui.entries) == 0 {
		emptyStyle := tcell.StyleDefault.Foreground(tcell.ColorGray)
		ui.drawText(2, listStartY, emptyStyle, "(empty directory)")
	} else {
		endIndex := ui.scrollOffset + viewableLines
		if endIndex > len(ui.entries) {
			endIndex = len(ui.entries)
		}

		for i := ui.scrollOffset; i < endIndex; i++ {
			y := listStartY + (i - ui.scrollOffset)
			entry := ui.entries[i]

			style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
			prefix := "  "

			if i == ui.selectedIndex {
				style = tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite).Bold(true)
				prefix = "► "
			}

			text := fmt.Sprintf("%s📁 %s", prefix, entry.Name())
			ui.drawText(2, y, style, text)
		}
	}

	// Status message
	if ui.statusMessage != "" {
		statusY := h - 3
		statusStyle := tcell.StyleDefault.Foreground(tcell.ColorRed)
		ui.drawText(2, statusY, statusStyle, ui.statusMessage)
	}

	// Footer
	footerStyle := tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorWhite)
	footerText := " [↑↓/jk]Navigate  [←/h]Parent  [→/l]Enter  [Enter]Select  [Esc]Cancel "
	ui.drawTextCentered(h-1, w, footerStyle, footerText)

	screen.Show()
}

func (ui *DirectoryBrowserUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *ui.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}

func (ui *DirectoryBrowserUI) drawTextCentered(y, width int, style tcell.Style, text string) {
	screen := *ui.screen
	x := (width - len(text)) / 2
	if x < 0 {
		x = 0
	}
	// Fill entire row with background
	for i := 0; i < width; i++ {
		screen.SetContent(i, y, ' ', nil, style)
	}
	// Draw text
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}
