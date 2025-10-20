package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"plexlinker/internal/config"
	"plexlinker/internal/downloader"
	"plexlinker/internal/logger"
)

// GeneratorMode represents the current interaction mode
type GeneratorMode int

const (
	ModePreview GeneratorMode = iota // Viewing the preview
)

type ScriptGeneratorState struct {
	config        *config.Config
	categorized   *downloader.CategorizedFiles
	selectedFiles []string // Full URLs
	mode          GeneratorMode
	tvShowPath    string   // Selected path for TV shows
	statusMessage string
	baseURL       string
	currentUser   string   // Current username for task naming
}

type ScriptGeneratorUI struct {
	screen *tcell.Screen
	state  *ScriptGeneratorState
}

func NewScriptGeneratorUI(screen *tcell.Screen, cfg *config.Config, selectedFiles []string, baseURL string, currentUser string) *ScriptGeneratorUI {
	// Extract just the filenames from the full URLs for categorization
	filenames := make([]string, len(selectedFiles))
	for i, url := range selectedFiles {
		// Extract filename from URL
		parts := strings.Split(url, "/")
		filenames[i] = parts[len(parts)-1]
	}

	categorized := downloader.CategorizeFiles(filenames)

	state := &ScriptGeneratorState{
		config:        cfg,
		categorized:   categorized,
		selectedFiles: selectedFiles,
		mode:          ModePreview,
		baseURL:       baseURL,
		tvShowPath:    cfg.TVShowsPath, // Default to config path
		currentUser:   currentUser,
	}

	return &ScriptGeneratorUI{
		screen: screen,
		state:  state,
	}
}

func (ui *ScriptGeneratorUI) HandleKeyEvent(ev *tcell.EventKey) bool {
	return ui.handlePreviewKeys(ev)
}

func (ui *ScriptGeneratorUI) handlePreviewKeys(ev *tcell.EventKey) bool {
	state := ui.state

	switch ev.Key() {
	case tcell.KeyEscape:
		return true // Exit back to browser
	case tcell.KeyEnter, tcell.KeyRune:
		if ev.Key() == tcell.KeyRune && ev.Rune() != '5' && ev.Rune() != '6' && ev.Rune() != 'b' {
			return false
		}
		// Handle key '5' or Enter - Generate & Run
		if ev.Key() == tcell.KeyEnter || ev.Rune() == '5' {
			_ = ui.generateAndRun(true) // true = execute scripts
			return true
		}
		// Handle key '6' - Generate Only (no execution)
		if ev.Rune() == '6' {
			_ = ui.generateAndRun(false) // false = don't execute
			return true
		}
		// Handle key 'b' - Browse TV folders
		if ev.Rune() == 'b' && state.categorized.HasTVShows() {
			ui.browseTVPath()
			return false
		}
	}
	return false
}

func (ui *ScriptGeneratorUI) browseTVPath() {
	state := ui.state
	browser := NewDirectoryBrowserUI(ui.screen, state.tvShowPath, "Select TV Shows Destination")

	for {
		browser.Draw()

		ev := (*ui.screen).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if result := browser.HandleKeyEvent(ev); result != nil {
				// User selected a path
				state.tvShowPath = *result
				state.statusMessage = fmt.Sprintf("TV shows path: %s", *result)
				return
			} else if ev.Key() == tcell.KeyEscape || (ev.Key() == tcell.KeyRune && ev.Rune() == 'q') {
				// User cancelled
				return
			}
		case *tcell.EventResize:
			(*ui.screen).Sync()
		}
	}
}


func (ui *ScriptGeneratorUI) generateAndRun(executeScripts bool) error {
	state := ui.state

	// Use the selected TV show path
	tvShowDest := state.tvShowPath

	// Build URL mapping: filename -> full URL
	urlMapping := make(map[string]string)
	for _, url := range state.selectedFiles {
		parts := strings.Split(url, "/")
		filename := parts[len(parts)-1]
		urlMapping[filename] = url
	}

	// Create script generator
	generator := downloader.NewScriptGenerator(state.config)

	// Generate all scripts
	scriptPaths, err := generator.GenerateCategorizedScripts(
		state.categorized,
		urlMapping,
		state.config.MoviesPath,
		tvShowDest,
		state.config.OtherPath,
	)
	if err != nil {
		state.statusMessage = fmt.Sprintf("Error: %v", err)
		return err
	}

	// Get script directory for status message
	scriptDir := filepath.Dir(scriptPaths[0])

	// If user chose "Generate Only", skip execution
	if !executeScripts {
		state.statusMessage = fmt.Sprintf("✓ Generated %d script(s) in %s (not executed)", len(scriptPaths), scriptDir)
		return nil
	}

	// Execute each script in a new window and track execution methods
	successCount := 0
	taskSchedulerCount := 0
	fallbackCount := 0
	var executions []*downloader.ScriptExecution

	for _, scriptPath := range scriptPaths {
		if scriptPath == "" {
			continue
		}

		logger.Get().Main().Printf("Executing script: %s", scriptPath)
		execution, err := generator.ExecuteScript(scriptPath, 1, state.currentUser)
		if err != nil {
			logger.Get().Main().Printf("Error executing script: %v", err)
			state.statusMessage = fmt.Sprintf("Error: %v", err)
			// Continue executing other scripts even if one fails
		} else {
			successCount++
			executions = append(executions, execution)

			// Track execution method
			if execution.ExecutionMethod == "task_scheduler" {
				taskSchedulerCount++
				logger.Get().Main().Printf("Successfully launched script via Task Scheduler: %s (task: %s)",
					filepath.Base(scriptPath), execution.TaskName)
			} else {
				fallbackCount++
				logger.Get().Main().Printf("Successfully launched script (direct): %s", filepath.Base(scriptPath))
			}
		}
	}

	// Build status message based on execution methods
	if fallbackCount == 0 && taskSchedulerCount > 0 {
		// All via Task Scheduler - best case
		state.statusMessage = fmt.Sprintf("✓ Launched %d via Task Scheduler (SSH-safe)", taskSchedulerCount)
	} else if fallbackCount > 0 && taskSchedulerCount > 0 {
		// Mixed - some fallback
		state.statusMessage = fmt.Sprintf("⚠ %d Task Scheduler, %d FALLBACK - Keep SSH open!", taskSchedulerCount, fallbackCount)
	} else if fallbackCount > 0 {
		// All fallback - worst case
		state.statusMessage = fmt.Sprintf("⚠ ALL %d IN FALLBACK MODE - DO NOT CLOSE SSH!", fallbackCount)
	} else {
		// Default (no executions or errors)
		state.statusMessage = fmt.Sprintf("Generated %d script(s) in %s", len(scriptPaths), scriptDir)
	}

	return nil
}

func (ui *ScriptGeneratorUI) Draw() {
	screen := *ui.screen
	screen.Clear()
	w, h := screen.Size()

	ui.drawPreview(w, h)

	screen.Show()
}

func (ui *ScriptGeneratorUI) drawPreview(w, h int) {
	state := ui.state

	// Header
	headerStyle := tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorWhite).Bold(true)
	headerText := " Script Generation Preview "
	ui.drawTextCentered(0, w, headerStyle, headerText)

	// Categorization results
	y := 2
	defaultStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)

	ui.drawText(2, y, defaultStyle.Bold(true), "Auto-Categorization Results:")
	y += 2

	// TV Shows
	if state.categorized.HasTVShows() {
		tvIcon := "📺"
		tvText := fmt.Sprintf("%s TV Shows: %d files → %s", tvIcon, len(state.categorized.TVShows), ui.getTVShowDestination())
		ui.drawText(4, y, tcell.StyleDefault.Foreground(tcell.ColorGreen), tvText)
		y++
	}

	// Movies
	if state.categorized.HasMovies() {
		movieIcon := "🎬"
		movieText := fmt.Sprintf("%s Movies: %d files → %s", movieIcon, len(state.categorized.Movies), state.config.MoviesPath)
		ui.drawText(4, y, tcell.StyleDefault.Foreground(tcell.ColorYellow), movieText)
		y++
	}

	// Other
	if state.categorized.HasOther() {
		otherIcon := "📁"
		otherText := fmt.Sprintf("%s Other: %d files → %s", otherIcon, len(state.categorized.Other), state.config.OtherPath)
		ui.drawText(4, y, tcell.StyleDefault.Foreground(tcell.ColorGray), otherText)
		y++
	}

	y += 2

	// TV Shows destination control
	if state.categorized.HasTVShows() {
		ui.drawText(2, y, defaultStyle.Bold(true), "TV Shows Destination:")
		y++
		destText := fmt.Sprintf("  Current: %s", ui.getTVShowDestination())
		ui.drawText(2, y, defaultStyle, destText)
		y++
		ui.drawText(2, y, tcell.StyleDefault.Foreground(tcell.ColorTeal), "  Press 'b' to browse folders or create new folder")
		y += 2
	}

	// Summary
	ui.drawText(2, y, defaultStyle.Bold(true), "Summary:")
	y++

	scriptCount := 0
	if state.categorized.HasTVShows() {
		scriptCount++
	}
	if state.categorized.HasMovies() {
		scriptCount++
	}
	if state.categorized.HasOther() {
		scriptCount++
	}

	summaryText := fmt.Sprintf("  %d script(s) will be generated and executed:", scriptCount)
	ui.drawText(2, y, defaultStyle, summaryText)
	y++

	if state.categorized.HasTVShows() {
		ui.drawText(4, y, defaultStyle, fmt.Sprintf("• plexhelper_tvshows.ps1 (%d files to %s)", len(state.categorized.TVShows), ui.getTVShowDestination()))
		y++
	}
	if state.categorized.HasMovies() {
		ui.drawText(4, y, defaultStyle, fmt.Sprintf("• plexhelper_movies.ps1 (%d files to %s)", len(state.categorized.Movies), state.config.MoviesPath))
		y++
	}
	if state.categorized.HasOther() {
		ui.drawText(4, y, defaultStyle, fmt.Sprintf("• plexhelper_other.ps1 (%d files to %s)", len(state.categorized.Other), state.config.OtherPath))
		y++
	}

	// Status message (if present, display it above the footer)
	if state.statusMessage != "" {
		y++
		// Color based on status message content
		statusStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		if strings.Contains(state.statusMessage, "✓") {
			statusStyle = tcell.StyleDefault.Foreground(tcell.ColorGreen)
		} else if strings.Contains(state.statusMessage, "⚠") || strings.Contains(state.statusMessage, "FALLBACK") {
			statusStyle = tcell.StyleDefault.Foreground(tcell.ColorYellow)
		} else if strings.Contains(state.statusMessage, "Error") {
			statusStyle = tcell.StyleDefault.Foreground(tcell.ColorRed)
		}
		ui.drawText(2, y, statusStyle, state.statusMessage)
	}

	// Footer
	footerStyle := tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorWhite)
	footerText := " [5]Generate & Run    [6]Generate Only    [B]rowse TV Folders    [Esc]Cancel "
	ui.drawTextCentered(h-1, w, footerStyle, footerText)
}

func (ui *ScriptGeneratorUI) getTVShowDestination() string {
	state := ui.state
	if state.tvShowPath == "" || state.tvShowPath == "auto" {
		return "auto (use Browse to select)"
	}
	return state.tvShowPath
}

func (ui *ScriptGeneratorUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *ui.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}

func (ui *ScriptGeneratorUI) drawTextCentered(y, width int, style tcell.Style, text string) {
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
