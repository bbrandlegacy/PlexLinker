package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"plexlinker/internal/tvrenamer/filesystem"
)

type TVBrowserState struct {
	browser       *filesystem.FileSystemBrowser
	listing       *filesystem.DirectoryListing
	cursorIndex   int
	scrollOffset  int
	selectedFiles map[string]*filesystem.FileItem
	statusMessage string
	isLoading     bool
	filterPattern string
}

type TVBrowserUI struct {
	screen *tcell.Screen
	state  *TVBrowserState
}

func NewTVBrowserUI(screen *tcell.Screen) *TVBrowserUI {
	return &TVBrowserUI{
		screen: screen,
		state: &TVBrowserState{
			browser:       filesystem.NewFileSystemBrowser(),
			selectedFiles: make(map[string]*filesystem.FileItem),
		},
	}
}

func (ui *TVBrowserUI) LoadCurrentDirectory() error {
	ui.state.isLoading = true
	ui.Draw()

	listing, err := ui.state.browser.ListDirectory()
	if err != nil {
		ui.state.isLoading = false
		ui.state.statusMessage = fmt.Sprintf("Error loading directory: %v", err)
		return err
	}

	ui.state.listing = listing
	ui.state.cursorIndex = 0
	ui.state.scrollOffset = 0
	ui.state.isLoading = false
	ui.state.statusMessage = fmt.Sprintf("Loaded %d items from %s", len(listing.Items), listing.CurrentPath)

	return nil
}

func (ui *TVBrowserUI) HandleKeyEvent(ev *tcell.EventKey) bool {
	if ui.state.listing == nil || len(ui.state.listing.Items) == 0 {
		return false
	}

	switch ev.Key() {
	case tcell.KeyUp:
		ui.moveCursorUp()
	case tcell.KeyDown:
		ui.moveCursorDown()
	case tcell.KeyEnter:
		return ui.handleEnter()
	case tcell.KeyBackspace:
		return ui.navigateUp()
	case tcell.KeyEscape:
		return ui.handleEscape()
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'k':
			ui.moveCursorUp()
		case 'j':
			ui.moveCursorDown()
		case 'h':
			return ui.navigateUp()
		case 'l':
			return ui.handleEnter()
		case '1':
			return ui.selectAllFiles()
		case '2':
			return ui.selectAllRecursive()
		case '3':
			ui.deselectAll()
		case '4':
			return ui.previewRenames()
		case '5':
			return ui.executeRenames()
		case '0':
			return true
		case 'q':
			return true
		}
	}

	return false
}

func (ui *TVBrowserUI) moveCursorUp() {
	if ui.state.cursorIndex > 0 {
		ui.state.cursorIndex--
		ui.adjustScrollForCursor()
	}
}

func (ui *TVBrowserUI) moveCursorDown() {
	if ui.state.cursorIndex < len(ui.state.listing.Items)-1 {
		ui.state.cursorIndex++
		ui.adjustScrollForCursor()
	}
}

func (ui *TVBrowserUI) adjustScrollForCursor() {
	_, h := (*ui.screen).Size()
	maxVisibleItems := h - 6

	if ui.state.cursorIndex < ui.state.scrollOffset {
		ui.state.scrollOffset = ui.state.cursorIndex
	}

	if ui.state.cursorIndex >= ui.state.scrollOffset+maxVisibleItems {
		ui.state.scrollOffset = ui.state.cursorIndex - maxVisibleItems + 1
	}

	if ui.state.scrollOffset < 0 {
		ui.state.scrollOffset = 0
	}
}

func (ui *TVBrowserUI) handleEnter() bool {
	if ui.state.cursorIndex >= len(ui.state.listing.Items) {
		return false
	}

	item := &ui.state.listing.Items[ui.state.cursorIndex]

	if item.IsDirectory {
		if err := ui.state.browser.NavigateInto(item.Name); err != nil {
			ui.state.statusMessage = fmt.Sprintf("Navigation error: %v", err)
		} else {
			ui.LoadCurrentDirectory()
		}
	} else if item.IsVideo {
		ui.toggleFileSelection(item)
		ui.moveCursorDown()
	} else {
		ui.state.statusMessage = "File is not a video file"
	}

	return false
}

func (ui *TVBrowserUI) navigateUp() bool {
	if err := ui.state.browser.NavigateUp(); err != nil {
		ui.state.statusMessage = fmt.Sprintf("Cannot navigate up: %v", err)
		return false
	}
	ui.LoadCurrentDirectory()
	return false
}

func (ui *TVBrowserUI) handleEscape() bool {
	return true // Exit to main menu
}

func (ui *TVBrowserUI) toggleFileSelection(item *filesystem.FileItem) {
	if _, exists := ui.state.selectedFiles[item.Path]; exists {
		delete(ui.state.selectedFiles, item.Path)
		ui.state.statusMessage = fmt.Sprintf("Deselected: %s", item.Name)
	} else {
		ui.state.selectedFiles[item.Path] = item
		ui.state.statusMessage = fmt.Sprintf("Selected: %s", item.Name)
	}
}

func (ui *TVBrowserUI) selectAllFiles() bool {
	count := 0
	for i := range ui.state.listing.Items {
		item := &ui.state.listing.Items[i]
		if item.IsVideo {
			ui.state.selectedFiles[item.Path] = item
			count++
		}
	}
	ui.state.statusMessage = fmt.Sprintf("Selected %d video files in current directory", count)
	return false
}

func (ui *TVBrowserUI) selectAllRecursive() bool {
	ui.state.isLoading = true
	ui.state.statusMessage = "Scanning directories recursively..."
	ui.Draw()

	videoFiles, err := ui.state.browser.GetVideoFilesRecursive()
	if err != nil {
		ui.state.isLoading = false
		ui.state.statusMessage = fmt.Sprintf("Error scanning recursively: %v", err)
		return false
	}

	count := 0
	for _, item := range videoFiles {
		ui.state.selectedFiles[item.Path] = item
		count++
	}

	ui.state.isLoading = false
	ui.state.statusMessage = fmt.Sprintf("Selected %d video files recursively", count)
	return false
}

func (ui *TVBrowserUI) deselectAll() {
	count := len(ui.state.selectedFiles)
	ui.state.selectedFiles = make(map[string]*filesystem.FileItem)
	ui.state.statusMessage = fmt.Sprintf("Deselected %d files", count)
}

func (ui *TVBrowserUI) previewRenames() bool {
	if len(ui.state.selectedFiles) == 0 {
		ui.state.statusMessage = "No files selected for preview"
		return false
	}

	return ui.showPreviewScreen()
}

func (ui *TVBrowserUI) executeRenames() bool {
	if len(ui.state.selectedFiles) == 0 {
		ui.state.statusMessage = "No files selected for renaming"
		return false
	}

	return ui.showExecuteScreen()
}

func (ui *TVBrowserUI) showPreviewScreen() bool {
	previewUI := NewPreviewUI(ui.screen, ui.state.selectedFiles)
	for {
		previewUI.Draw()

		ev := (*ui.screen).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if previewUI.HandleKeyEvent(ev) {
				return false // Exit preview
			}
		case *tcell.EventResize:
			(*ui.screen).Sync()
		}
	}
}

func (ui *TVBrowserUI) showExecuteScreen() bool {
	executeUI := NewExecuteUI(ui.screen, ui.state.selectedFiles)
	for {
		executeUI.Draw()

		ev := (*ui.screen).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if executeUI.HandleKeyEvent(ev) {
				// Refresh the directory listing after rename operations
				ui.LoadCurrentDirectory()
				ui.deselectAll()
				return false
			}
		case *tcell.EventResize:
			(*ui.screen).Sync()
		}
	}
}

func (ui *TVBrowserUI) Draw() {
	screen := *ui.screen
	w, h := screen.Size()
	screen.Clear()

	if ui.state.isLoading {
		ui.drawLoading(w, h)
		screen.Show()
		return
	}

	ui.drawHeader(w)
	ui.drawDirectoryListing(w, h)
	ui.drawStatusBar(w, h)

	screen.Show()
}

func (ui *TVBrowserUI) drawLoading(w, h int) {
	message := "Loading directory..."
	x := (w - len(message)) / 2
	y := h / 2

	ui.drawText(x, y, tcell.StyleDefault.Foreground(tcell.ColorYellow), message)
}

func (ui *TVBrowserUI) drawHeader(w int) {
	title := "PlexLinker - TV Show Renamer"
	ui.drawText(0, 0, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite), title)

	if ui.state.listing != nil {
		pathText := fmt.Sprintf("Path: %s", ui.state.listing.CurrentPath)
		if len(pathText) > w {
			pathText = "..." + pathText[len(pathText)-w+3:]
		}
		ui.drawText(0, 1, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), pathText)
	}

	selectedCount := len(ui.state.selectedFiles)
	selectedText := fmt.Sprintf("Selected: %d files", selectedCount)
	ui.drawText(w-len(selectedText), 1, tcell.StyleDefault.Foreground(tcell.ColorGreen), selectedText)
}

func (ui *TVBrowserUI) drawDirectoryListing(w, h int) {
	if ui.state.listing == nil {
		return
	}

	startY := 3
	maxItems := h - 6

	for i := 0; i < maxItems; i++ {
		itemIndex := i + ui.state.scrollOffset
		if itemIndex >= len(ui.state.listing.Items) {
			break
		}

		item := ui.state.listing.Items[itemIndex]
		y := startY + i
		style := tcell.StyleDefault

		if itemIndex == ui.state.cursorIndex {
			style = style.Background(tcell.ColorNavy)
		}

		icon := ui.state.browser.GetFileIcon(&item)
		var nameColor tcell.Color

		if item.IsDirectory {
			nameColor = tcell.ColorBlue
		} else if item.IsVideo {
			if item.HasError {
				nameColor = tcell.ColorRed
			} else if item.TVInfo != nil {
				nameColor = tcell.ColorGreen
			} else {
				nameColor = tcell.ColorYellow
			}
		} else {
			nameColor = tcell.ColorWhite
		}

		style = style.Foreground(nameColor)

		if _, selected := ui.state.selectedFiles[item.Path]; selected {
			icon = "✓"
			style = style.Foreground(tcell.ColorGreen)
		}

		text := fmt.Sprintf("%s %s", icon, item.Name)
		if len(text) > w-2 {
			text = text[:w-5] + "..."
		}

		ui.drawText(1, y, style, text)

		// Show additional info for video files
		if item.IsVideo && itemIndex == ui.state.cursorIndex {
			infoText := ui.state.browser.GetFileInfo(&item)
			if len(infoText) > w-4 {
				infoText = infoText[:w-7] + "..."
			}
			ui.drawText(3, y+1, tcell.StyleDefault.Foreground(tcell.ColorLightGray), infoText)
		}
	}

	// Draw scroll indicator
	if len(ui.state.listing.Items) > maxItems {
		scrollInfo := fmt.Sprintf("(%d-%d of %d)",
			ui.state.scrollOffset+1,
			ui.state.scrollOffset+maxItems,
			len(ui.state.listing.Items))
		if ui.state.scrollOffset+maxItems > len(ui.state.listing.Items) {
			scrollInfo = fmt.Sprintf("(%d-%d of %d)",
				ui.state.scrollOffset+1,
				len(ui.state.listing.Items),
				len(ui.state.listing.Items))
		}

		ui.drawText(w-len(scrollInfo), startY-1, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), scrollInfo)
	}
}

func (ui *TVBrowserUI) drawStatusBar(w, h int) {
	statusY := h - 2
	commandsY := h - 1

	statusText := ui.state.statusMessage
	if statusText == "" {
		statusText = "Use arrow keys to navigate, Enter to select files/directories"
	}

	if len(statusText) > w {
		statusText = statusText[:w-3] + "..."
	}
	ui.drawText(0, statusY, tcell.StyleDefault.Foreground(tcell.ColorYellow), statusText)

	commands := "1:SelectAll 2:Recursive 3:Clear 4:Preview 5:Rename 0:Exit"
	if len(commands) > w {
		commands = "1:All 2:Rec 3:Clear 4:Preview 5:Rename 0:Exit"
	}

	ui.drawText(0, commandsY, tcell.StyleDefault.Foreground(tcell.ColorWhite), commands)
}

func (ui *TVBrowserUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *ui.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}