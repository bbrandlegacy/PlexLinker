package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"plexlinker/internal/browser"
	"plexlinker/internal/client"
	"plexlinker/internal/config"
	"plexlinker/internal/logger"
)

type BrowserState struct {
	listing           *browser.DirectoryListing
	cursorIndex       int
	scrollOffset      int
	selectedFiles     map[string]*browser.DirectoryItem
	client            *client.Client
	config            *config.Config
	statusMessage     string
	isLoading         bool
	loadingMessage    string
	cursorHistory     map[string]int // Maps directory path to cursor position
	lastScriptPID     int            // PID of last launched script
	lastScriptRunning bool           // Whether last script is still running
	currentUser       string         // Current username for task naming
}

type BrowserUI struct {
	screen *tcell.Screen
	state  *BrowserState
}

func NewBrowserUI(screen *tcell.Screen, cfg *config.Config, currentUser string) *BrowserUI {
	return &BrowserUI{
		screen: screen,
		state: &BrowserState{
			selectedFiles: make(map[string]*browser.DirectoryItem),
			client:        client.NewClient(cfg),
			config:        cfg,
			cursorHistory: make(map[string]int),
			currentUser:   currentUser,
		},
	}
}

func (ui *BrowserUI) LoadDirectory(path string) error {
	logger.Get().UI().Printf("LoadDirectory: Starting to load path='%s'", path)

	// Save current cursor position before changing directories
	if ui.state.listing != nil {
		ui.state.cursorHistory[ui.state.listing.CurrentPath] = ui.state.cursorIndex
	}

	ui.state.isLoading = true
	ui.state.loadingMessage = "Loading directory..."
	ui.Draw()

	logger.Get().UI().Printf("LoadDirectory: Calling client.FetchDirectory")
	html, err := ui.state.client.FetchDirectory(path)
	if err != nil {
		logger.Get().UI().Printf("LoadDirectory: FetchDirectory failed: %v", err)
		ui.state.isLoading = false
		ui.state.statusMessage = fmt.Sprintf("Error: %v", err)
		return err
	}

	logger.Get().UI().Printf("LoadDirectory: Parsing HTML (length=%d bytes)", len(html))

	// Only sort by date if at root level (empty path or base URL)
	isRoot := path == "" || ui.isAtBaseURL()
	listing, err := browser.ParseHTMLDirectoryWithSort(html, path, isRoot)
	if err != nil {
		logger.Get().UI().Printf("LoadDirectory: Parse failed: %v", err)
		ui.state.isLoading = false
		ui.state.statusMessage = fmt.Sprintf("Parse error: %v", err)
		return err
	}

	logger.Get().UI().Printf("LoadDirectory: Successfully loaded %d items", len(listing.Items))
	ui.state.listing = listing

	// Restore cursor position if we've been to this directory before
	if savedCursor, exists := ui.state.cursorHistory[path]; exists {
		ui.state.cursorIndex = savedCursor
		if ui.state.cursorIndex >= len(listing.Items) {
			ui.state.cursorIndex = len(listing.Items) - 1
		}
		if ui.state.cursorIndex < 0 {
			ui.state.cursorIndex = 0
		}
	} else {
		ui.state.cursorIndex = 0
	}

	ui.state.scrollOffset = 0
	ui.adjustScrollForCursor()
	ui.state.isLoading = false
	ui.state.statusMessage = fmt.Sprintf("Loaded %d items", len(listing.Items))

	return nil
}

func (ui *BrowserUI) HandleKeyEvent(ev *tcell.EventKey) bool {
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
			return ui.selectAllCurrent()
		case '2':
			return ui.selectAllRecursive()
		case '3':
			ui.deselectAll()
		case '4':
			return ui.showSelectedFiles()
		case '5':
			return ui.generateScript()
		case '0':
			return true
		}
	}

	return false
}

func (ui *BrowserUI) moveCursorUp() {
	if ui.state.cursorIndex > 0 {
		ui.state.cursorIndex--
		ui.adjustScrollForCursor()
	}
}

func (ui *BrowserUI) moveCursorDown() {
	if ui.state.cursorIndex < len(ui.state.listing.Items)-1 {
		ui.state.cursorIndex++
		ui.adjustScrollForCursor()
	}
}

func (ui *BrowserUI) adjustScrollForCursor() {
	_, h := (*ui.screen).Size()
	maxVisibleItems := h - 5 // Account for header and status bar

	// If cursor is above visible area, scroll up
	if ui.state.cursorIndex < ui.state.scrollOffset {
		ui.state.scrollOffset = ui.state.cursorIndex
	}

	// If cursor is below visible area, scroll down
	if ui.state.cursorIndex >= ui.state.scrollOffset+maxVisibleItems {
		ui.state.scrollOffset = ui.state.cursorIndex - maxVisibleItems + 1
	}

	// Ensure scroll offset doesn't go negative
	if ui.state.scrollOffset < 0 {
		ui.state.scrollOffset = 0
	}
}

func (ui *BrowserUI) handleEnter() bool {
	if ui.state.cursorIndex >= len(ui.state.listing.Items) {
		return false
	}

	item := &ui.state.listing.Items[ui.state.cursorIndex]

	if item.Type == browser.TypeDirectory {
		ui.LoadDirectory(item.Path)
	} else {
		ui.toggleFileSelection(item)
		ui.moveCursorDown()
	}

	return false
}

func (ui *BrowserUI) navigateUp() bool {
	parentPath := browser.GetParentPath(ui.state.listing.CurrentPath)

	// Check if we're at the base URL - don't allow going above it
	if ui.isAtBaseURL() {
		return false
	}

	ui.LoadDirectory(parentPath)
	return false
}

func (ui *BrowserUI) handleEscape() bool {
	// If we're at the base URL, go back to main menu
	if ui.isAtBaseURL() {
		return true
	}

	// Otherwise, navigate up one directory
	ui.navigateUp()
	return false
}

func (ui *BrowserUI) isAtBaseURL() bool {
	baseURL := ui.state.config.BaseURL
	if strings.HasSuffix(baseURL, "/") {
		baseURL = strings.TrimSuffix(baseURL, "/")
	}

	// If current path is empty (at root), we're at base
	if ui.state.listing.CurrentPath == "" {
		return true
	}

	// Build the full URL for current path to compare with base
	currentFullPath := ui.state.client.GetFullURL(ui.state.listing.CurrentPath)

	// Remove trailing slash from current path for comparison
	if strings.HasSuffix(currentFullPath, "/") {
		currentFullPath = strings.TrimSuffix(currentFullPath, "/")
	}

	return currentFullPath == baseURL
}

func (ui *BrowserUI) toggleFileSelection(item *browser.DirectoryItem) {
	fullPath := ui.state.client.GetFullURL(item.Path)

	if _, exists := ui.state.selectedFiles[fullPath]; exists {
		delete(ui.state.selectedFiles, fullPath)
		ui.state.statusMessage = fmt.Sprintf("Deselected: %s", item.Name)
	} else {
		ui.state.selectedFiles[fullPath] = item
		ui.state.statusMessage = fmt.Sprintf("Selected: %s", item.Name)
	}
}

func (ui *BrowserUI) selectAllCurrent() bool {
	count := 0
	for i := range ui.state.listing.Items {
		item := &ui.state.listing.Items[i]
		if item.Type == browser.TypeFile && (item.IsVideoFile() || item.IsAudioFile()) {
			fullPath := ui.state.client.GetFullURL(item.Path)
			ui.state.selectedFiles[fullPath] = item
			count++
		}
	}
	ui.state.statusMessage = fmt.Sprintf("Selected %d files in current directory", count)
	return false
}

func (ui *BrowserUI) selectAllRecursive() bool {
	ui.state.isLoading = true
	ui.state.loadingMessage = "Scanning directories recursively... (Press ESC to cancel)"
	ui.Draw()

	// Select all files in current directory first
	initialCount := 0
	for i := range ui.state.listing.Items {
		item := &ui.state.listing.Items[i]
		if item.Type == browser.TypeFile && (item.IsVideoFile() || item.IsAudioFile()) {
			fullPath := ui.state.client.GetFullURL(item.Path)
			if _, exists := ui.state.selectedFiles[fullPath]; !exists {
				ui.state.selectedFiles[fullPath] = item
				initialCount++
			}
		}
	}

	// Use channels for background processing
	resultChan := make(chan int, 1)
	progressChan := make(chan int, 10)
	cancelChan := make(chan bool, 1)

	// Start recursive scan in background goroutine
	go func() {
		count := ui.selectRecursiveHelperWithProgress(ui.state.listing.CurrentPath, progressChan, cancelChan)
		resultChan <- count
	}()

	// Event loop to handle progress updates and cancellation
	fileCount := initialCount
	for {
		select {
		case additionalCount := <-resultChan:
			// Scan completed
			ui.state.isLoading = false
			totalCount := initialCount + additionalCount
			ui.state.statusMessage = fmt.Sprintf("Selected %d files recursively", totalCount)
			return false

		case progress := <-progressChan:
			// Update progress
			fileCount += progress
			ui.state.loadingMessage = fmt.Sprintf("Scanning... Found %d files (Press ESC to cancel)", fileCount)
			ui.Draw()

		default:
			// Check for ESC key to cancel
			ev := (*ui.screen).PollEvent()
			if ev != nil {
				switch ev := ev.(type) {
				case *tcell.EventKey:
					if ev.Key() == tcell.KeyEscape {
						cancelChan <- true
						ui.state.isLoading = false
						ui.state.statusMessage = fmt.Sprintf("Scan cancelled. Selected %d files", fileCount)
						return false
					}
				case *tcell.EventResize:
					(*ui.screen).Sync()
				}
			}
		}
	}
}

func (ui *BrowserUI) selectRecursiveHelper(parentPath string) int {
	// Legacy blocking version - kept for compatibility
	return ui.selectRecursiveHelperWithProgress(parentPath, nil, nil)
}

func (ui *BrowserUI) selectRecursiveHelperWithProgress(parentPath string, progressChan chan<- int, cancelChan <-chan bool) int {
	// Check for cancellation
	select {
	case <-cancelChan:
		return 0
	default:
	}

	// Fetch the parent directory to get subdirectories
	html, err := ui.state.client.FetchDirectory(parentPath)
	if err != nil {
		return 0
	}

	listing, err := browser.ParseHTMLDirectory(html, parentPath)
	if err != nil {
		return 0
	}

	count := 0

	// Process each subdirectory
	for _, item := range listing.Items {
		// Check for cancellation before each operation
		select {
		case <-cancelChan:
			return count
		default:
		}

		if item.Type == browser.TypeDirectory {
			// Fetch this subdirectory
			subHTML, err := ui.state.client.FetchDirectory(item.Path)
			if err != nil {
				continue
			}

			subListing, err := browser.ParseHTMLDirectory(subHTML, item.Path)
			if err != nil {
				continue
			}

			// Select all video/audio files in this subdirectory
			batchCount := 0
			for _, subItem := range subListing.Items {
				if subItem.Type == browser.TypeFile && (subItem.IsVideoFile() || subItem.IsAudioFile()) {
					fullPath := ui.state.client.GetFullURL(subItem.Path)
					if _, exists := ui.state.selectedFiles[fullPath]; !exists {
						itemCopy := subItem
						ui.state.selectedFiles[fullPath] = &itemCopy
						batchCount++
					}
				}
			}

			// Report progress if channel provided
			if progressChan != nil && batchCount > 0 {
				select {
				case progressChan <- batchCount:
				default:
				}
			}
			count += batchCount

			// Recurse into subdirectories of this directory
			count += ui.selectRecursiveHelperWithProgress(item.Path, progressChan, cancelChan)
		}
	}

	return count
}

func (ui *BrowserUI) deselectAll() {
	count := len(ui.state.selectedFiles)
	ui.state.selectedFiles = make(map[string]*browser.DirectoryItem)
	ui.state.statusMessage = fmt.Sprintf("Deselected %d files", count)
}

func (ui *BrowserUI) generateScript() bool {
	if len(ui.state.selectedFiles) == 0 {
		ui.state.statusMessage = "No files selected"
		return false
	}

	// Collect selected file URLs
	selectedURLs := make([]string, 0, len(ui.state.selectedFiles))
	for url := range ui.state.selectedFiles {
		selectedURLs = append(selectedURLs, url)
	}

	// Create and show Script Generation Window
	generatorUI := NewScriptGeneratorUI(ui.screen, ui.state.config, selectedURLs, ui.state.config.BaseURL, ui.state.currentUser)

	for {
		// Draw the Script Generation Window
		generatorUI.Draw()

		// Handle events
		ev := (*ui.screen).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if shouldExit := generatorUI.HandleKeyEvent(ev); shouldExit {
				// User exited the generator, return to browser
				ui.state.statusMessage = generatorUI.state.statusMessage
				return false
			}
		case *tcell.EventResize:
			(*ui.screen).Sync()
		}
	}
}

func (ui *BrowserUI) showMediaTypeModal() *MediaType {
	modal := NewMediaTypeModalUI(ui.screen)

	for {
		// Redraw the browser UI first
		ui.Draw()
		// Then draw modal on top
		modal.Draw()

		ev := (*ui.screen).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if result := modal.HandleKeyEvent(ev); result != nil {
				return result
			}
		case *tcell.EventResize:
			(*ui.screen).Sync()
		}
	}
}

func (ui *BrowserUI) showShowSelector() *string {
	selector := NewShowSelectorUI(ui.screen)

	// Attempt to detect show name from selected file paths
	detectedShows := make(map[string]bool)
	for url := range ui.state.selectedFiles {
		if showName := ParseShowNameFromURL(url); showName != "" {
			detectedShows[showName] = true
		}
	}

	// Create show list (could be extended to read from W:\shows if accessible)
	shows := []string{}
	for show := range detectedShows {
		shows = append(shows, show)
	}

	// If no shows detected, provide a manual input option
	if len(shows) == 0 {
		shows = []string{"[Enter show name manually]"}
	}

	selector.SetManualShows(shows)

	for {
		// Redraw the browser UI first
		ui.Draw()
		// Then draw selector on top
		selector.Draw()

		ev := (*ui.screen).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if result := selector.HandleKeyEvent(ev); result != nil {
				return result
			}
		case *tcell.EventResize:
			(*ui.screen).Sync()
		}
	}
}

func (ui *BrowserUI) showConfirmationModal(message string) bool {
	modal := NewConfirmModalUI(ui.screen, message)

	for {
		// Redraw the browser UI first
		ui.Draw()
		// Then draw modal on top
		modal.Draw()

		ev := (*ui.screen).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if result := modal.HandleKeyEvent(ev); result != nil {
				return *result
			}
		case *tcell.EventResize:
			(*ui.screen).Sync()
		}
	}
}


func (ui *BrowserUI) showScriptViewer(scriptPath, scriptContent string) {
	scriptViewer := NewScriptViewerUI(ui.screen, scriptPath, scriptContent)

	for {
		scriptViewer.Draw()

		ev := (*ui.screen).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if scriptViewer.HandleKeyEvent(ev) {
				return // Exit script viewer
			}
		case *tcell.EventResize:
			(*ui.screen).Sync()
		}
	}
}


func (ui *BrowserUI) showSelectedFiles() bool {
	if len(ui.state.selectedFiles) == 0 {
		ui.state.statusMessage = "No files selected"
		return false
	}

	viewer := NewSelectedViewerUI(ui.screen, ui.state.selectedFiles)

	for {
		viewer.Draw()

		ev := (*ui.screen).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if viewer.HandleKeyEvent(ev) {
				// Returned to browser - redraw current state
				return false
			}
		case *tcell.EventResize:
			(*ui.screen).Sync()
		}
	}
}

func (ui *BrowserUI) Draw() {
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

func (ui *BrowserUI) drawLoading(w, h int) {
	message := ui.state.loadingMessage
	if message == "" {
		message = "Loading..."
	}

	x := (w - len(message)) / 2
	y := h / 2

	ui.drawText(x, y, tcell.StyleDefault.Foreground(tcell.ColorYellow), message)
}

func (ui *BrowserUI) drawHeader(w int) {
	title := "PlexLinker - Directory Browser"
	ui.drawText(0, 0, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite), title)

	if ui.state.listing != nil {
		path := ui.state.listing.CurrentPath
		if path == "" {
			path = "/"
		}
		pathText := fmt.Sprintf("Path: %s", path)
		ui.drawText(0, 1, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), pathText)
	}

	selectedCount := len(ui.state.selectedFiles)
	selectedText := fmt.Sprintf("Selected: %d files", selectedCount)
	ui.drawText(w-len(selectedText), 1, tcell.StyleDefault.Foreground(tcell.ColorGreen), selectedText)
}

func (ui *BrowserUI) drawDirectoryListing(w, h int) {
	if ui.state.listing == nil {
		return
	}

	startY := 3
	maxItems := h - 5

	// Draw visible items based on scroll offset
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

		icon := item.GetFileIcon()
		if item.Type == browser.TypeDirectory {
			style = style.Foreground(tcell.ColorBlue)
		} else {
			style = style.Foreground(tcell.ColorWhite)
		}

		fullPath := ui.state.client.GetFullURL(item.Path)
		if _, selected := ui.state.selectedFiles[fullPath]; selected {
			icon = "✓"
			style = style.Foreground(tcell.ColorGreen)
		}

		text := fmt.Sprintf("%s %s", icon, item.Name)
		if len(text) > w-2 {
			text = text[:w-5] + "..."
		}

		ui.drawText(1, y, style, text)
	}

	// Draw scroll indicator if needed
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

func (ui *BrowserUI) drawStatusBar(w, h int) {
	statusY := h - 2
	commandsY := h - 1

	statusText := ui.state.statusMessage
	if statusText == "" {
		statusText = "Use arrow keys or hjkl to navigate"
	}

	ui.drawText(0, statusY, tcell.StyleDefault.Foreground(tcell.ColorYellow), statusText)

	commands := "1:SelectCurrent 2:SelectAll(R) 3:Deselect 4:ShowSelected 5:GenScript 0:Exit"
	if len(commands) > w {
		commands = "1:Cur 2:All(R) 3:Clear 4:Show 5:Script 0:Exit"
	}

	ui.drawText(0, commandsY, tcell.StyleDefault.Foreground(tcell.ColorWhite), commands)
}

func (ui *BrowserUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *ui.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}
