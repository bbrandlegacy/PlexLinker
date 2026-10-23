package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"plexlinker/internal/tvrenamer/filesystem"
	"plexlinker/internal/tvrenamer/patterns"
)

type RenameResult struct {
	OriginalPath string
	NewPath      string
	Success      bool
	Error        string
}

type ExecuteUI struct {
	screen        *tcell.Screen
	selectedFiles map[string]*filesystem.FileItem
	results       []RenameResult
	scrollOffset  int
	isExecuting   bool
	isCompleted   bool
	statusMessage string
	undoLog       []RenameResult
}

func NewExecuteUI(screen *tcell.Screen, selectedFiles map[string]*filesystem.FileItem) *ExecuteUI {
	return &ExecuteUI{
		screen:        screen,
		selectedFiles: selectedFiles,
		statusMessage: fmt.Sprintf("Ready to rename %d files", len(selectedFiles)),
	}
}

func (ui *ExecuteUI) HandleKeyEvent(ev *tcell.EventKey) bool {
	if ui.isExecuting {
		return false // Don't allow input during execution
	}

	_, h := (*ui.screen).Size()
	maxVisibleItems := h - 6

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
		case 'r':
			if !ui.isCompleted {
				ui.executeRenames()
			}
		case 'u':
			if ui.isCompleted && len(ui.undoLog) > 0 {
				ui.undoRenames()
			}
		case 'l':
			if ui.isCompleted {
				ui.saveLogFile()
			}
		}
	}

	return false
}

func (ui *ExecuteUI) scrollUp() {
	if ui.scrollOffset > 0 {
		ui.scrollOffset--
	}
}

func (ui *ExecuteUI) scrollDown(maxVisible int) {
	maxItems := len(ui.results)
	if !ui.isCompleted {
		maxItems = len(ui.selectedFiles)
	}

	maxScroll := maxItems - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if ui.scrollOffset < maxScroll {
		ui.scrollOffset++
	}
}

func (ui *ExecuteUI) scrollPageUp(maxVisible int) {
	ui.scrollOffset -= maxVisible / 2
	if ui.scrollOffset < 0 {
		ui.scrollOffset = 0
	}
}

func (ui *ExecuteUI) scrollPageDown(maxVisible int) {
	maxItems := len(ui.results)
	if !ui.isCompleted {
		maxItems = len(ui.selectedFiles)
	}

	maxScroll := maxItems - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	ui.scrollOffset += maxVisible / 2
	if ui.scrollOffset > maxScroll {
		ui.scrollOffset = maxScroll
	}
}

func (ui *ExecuteUI) scrollToEnd(maxVisible int) {
	maxItems := len(ui.results)
	if !ui.isCompleted {
		maxItems = len(ui.selectedFiles)
	}

	maxScroll := maxItems - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	ui.scrollOffset = maxScroll
}

func (ui *ExecuteUI) executeRenames() {
	ui.isExecuting = true
	ui.statusMessage = "Executing renames..."

	plexNamer := patterns.NewPlexNamer(nil)
	ui.results = make([]RenameResult, 0, len(ui.selectedFiles))

	// Sort files for consistent execution order
	var files []*filesystem.FileItem
	for _, item := range ui.selectedFiles {
		files = append(files, item)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	for _, item := range files {
		ui.Draw() // Update progress display

		result := RenameResult{
			OriginalPath: item.Path,
		}

		if item.TVInfo == nil {
			result.Success = false
			result.Error = "No TV show information parsed"
			result.NewPath = item.Path
		} else {
			newFilename := plexNamer.GeneratePlexName(item.TVInfo)
			newPath := filepath.Join(filepath.Dir(item.Path), newFilename)
			result.NewPath = newPath

			if err := os.Rename(item.Path, newPath); err != nil {
				result.Success = false
				result.Error = err.Error()
			} else {
				result.Success = true
				ui.undoLog = append(ui.undoLog, result)
			}
		}

		ui.results = append(ui.results, result)
	}

	ui.isExecuting = false
	ui.isCompleted = true

	successCount := 0
	for _, result := range ui.results {
		if result.Success {
			successCount++
		}
	}

	ui.statusMessage = fmt.Sprintf("Completed: %d/%d files renamed successfully", successCount, len(ui.results))
}

func (ui *ExecuteUI) undoRenames() {
	ui.isExecuting = true
	ui.statusMessage = "Undoing renames..."

	undoCount := 0
	for i := len(ui.undoLog) - 1; i >= 0; i-- {
		result := ui.undoLog[i]
		if err := os.Rename(result.NewPath, result.OriginalPath); err == nil {
			undoCount++
		}
	}

	ui.undoLog = []RenameResult{}
	ui.isExecuting = false
	ui.statusMessage = fmt.Sprintf("Undone: %d renames reverted", undoCount)
}

func (ui *ExecuteUI) saveLogFile() {
	timestamp := time.Now().Format("20060102_150405")
	logPath := fmt.Sprintf("plexlinker_rename_log_%s.txt", timestamp)

	var logContent strings.Builder
	fmt.Fprintf(&logContent, "PlexLinker Rename Log - %s\n", time.Now().Format("2006-01-02 15:04:05"))
	logContent.WriteString(strings.Repeat("=", 50) + "\n\n")

	for _, result := range ui.results {
		status := "SUCCESS"
		if !result.Success {
			status = "FAILED"
		}

		fmt.Fprintf(&logContent, "[%s] %s\n", status, filepath.Base(result.OriginalPath))
		fmt.Fprintf(&logContent, "  From: %s\n", result.OriginalPath)
		fmt.Fprintf(&logContent, "  To:   %s\n", result.NewPath)
		if result.Error != "" {
			fmt.Fprintf(&logContent, "  Error: %s\n", result.Error)
		}
		logContent.WriteString("\n")
	}

	if err := os.WriteFile(logPath, []byte(logContent.String()), 0644); err != nil {
		ui.statusMessage = fmt.Sprintf("Failed to save log: %v", err)
	} else {
		ui.statusMessage = fmt.Sprintf("Log saved: %s", logPath)
	}
}

func (ui *ExecuteUI) Draw() {
	screen := *ui.screen
	w, h := screen.Size()
	screen.Clear()

	ui.drawHeader(w)
	ui.drawExecutionList(w, h)
	ui.drawStatusBar(w, h)

	screen.Show()
}

func (ui *ExecuteUI) drawHeader(w int) {
	title := "PlexLinker - Execute Renames"
	ui.drawText(0, 0, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite), title)

	var subtitle string
	if ui.isExecuting {
		subtitle = "Renaming files in progress..."
	} else if ui.isCompleted {
		successCount := 0
		for _, result := range ui.results {
			if result.Success {
				successCount++
			}
		}
		subtitle = fmt.Sprintf("Completed: %d/%d files renamed", successCount, len(ui.results))
	} else {
		subtitle = fmt.Sprintf("Ready to rename %d files", len(ui.selectedFiles))
	}

	ui.drawText(0, 1, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), subtitle)
}

func (ui *ExecuteUI) drawExecutionList(w, h int) {
	startY := 3
	maxItems := h - 6

	if ui.isCompleted {
		// Show results
		for i := 0; i < maxItems && i+ui.scrollOffset < len(ui.results); i++ {
			itemIndex := i + ui.scrollOffset
			result := ui.results[itemIndex]
			y := startY + i

			var style tcell.Style
			var indicator string

			if result.Success {
				style = tcell.StyleDefault.Foreground(tcell.ColorGreen)
				indicator = "✅"
			} else {
				style = tcell.StyleDefault.Foreground(tcell.ColorRed)
				indicator = "❌"
			}

			text := fmt.Sprintf("%s %s → %s", indicator, filepath.Base(result.OriginalPath), filepath.Base(result.NewPath))
			if result.Error != "" {
				text += fmt.Sprintf(" (%s)", result.Error)
			}

			if len(text) > w-2 {
				text = text[:w-5] + "..."
			}

			ui.drawText(1, y, style, text)
		}

		// Draw scroll indicator
		if len(ui.results) > maxItems {
			scrollInfo := fmt.Sprintf("(%d-%d of %d)",
				ui.scrollOffset+1,
				min(ui.scrollOffset+maxItems, len(ui.results)),
				len(ui.results))

			ui.drawText(w-len(scrollInfo), startY-1, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), scrollInfo)
		}
	} else {
		// Show files to be renamed
		plexNamer := patterns.NewPlexNamer(nil)
		var files []*filesystem.FileItem
		for _, item := range ui.selectedFiles {
			files = append(files, item)
		}
		sort.Slice(files, func(i, j int) bool {
			return files[i].Path < files[j].Path
		})

		for i := 0; i < maxItems && i+ui.scrollOffset < len(files); i++ {
			itemIndex := i + ui.scrollOffset
			item := files[itemIndex]
			y := startY + i

			var style tcell.Style
			var text string

			if item.TVInfo != nil {
				newName := plexNamer.GeneratePlexName(item.TVInfo)
				text = fmt.Sprintf("📁 %s → %s", item.Name, newName)
				style = tcell.StyleDefault.Foreground(tcell.ColorYellow)
			} else {
				text = fmt.Sprintf("❌ %s (cannot parse)", item.Name)
				style = tcell.StyleDefault.Foreground(tcell.ColorRed)
			}

			if len(text) > w-2 {
				text = text[:w-5] + "..."
			}

			ui.drawText(1, y, style, text)
		}

		// Draw scroll indicator
		if len(files) > maxItems {
			scrollInfo := fmt.Sprintf("(%d-%d of %d)",
				ui.scrollOffset+1,
				min(ui.scrollOffset+maxItems, len(files)),
				len(files))

			ui.drawText(w-len(scrollInfo), startY-1, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), scrollInfo)
		}
	}
}

func (ui *ExecuteUI) drawStatusBar(w, h int) {
	statusY := h - 2
	commandsY := h - 1

	statusText := ui.statusMessage
	if len(statusText) > w {
		statusText = statusText[:w-3] + "..."
	}

	var statusColor tcell.Color
	if ui.isExecuting {
		statusColor = tcell.ColorYellow
	} else if ui.isCompleted {
		statusColor = tcell.ColorGreen
	} else {
		statusColor = tcell.ColorLightCyan
	}

	ui.drawText(0, statusY, tcell.StyleDefault.Foreground(statusColor), statusText)

	var commands string
	if ui.isExecuting {
		commands = "Please wait... renaming in progress"
	} else if ui.isCompleted {
		if len(ui.undoLog) > 0 {
			commands = "u:Undo  l:SaveLog  ↑↓:Scroll  Esc:Exit"
		} else {
			commands = "l:SaveLog  ↑↓:Scroll  Esc:Exit"
		}
	} else {
		commands = "r:Execute  ↑↓:Scroll  Esc:Cancel"
	}

	if len(commands) > w {
		if ui.isCompleted {
			commands = "u:Undo l:Log Esc:Exit"
		} else {
			commands = "r:Execute Esc:Cancel"
		}
	}

	ui.drawText(0, commandsY, tcell.StyleDefault.Foreground(tcell.ColorWhite), commands)
}

func (ui *ExecuteUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *ui.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}
