package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"plexlinker/internal/config"
)

type SettingsField int

const (
	FieldBaseURL SettingsField = iota
	FieldUsername
	FieldPassword
	FieldMoviesPath
	FieldTVShowsPath
	FieldOtherPath
	FieldScriptFormat
	FieldDebugMode
	FieldSave
	FieldCancel
)

type SettingsUI struct {
	screen        *tcell.Screen
	config        *config.Config
	tempConfig    *config.Config
	currentField  SettingsField
	statusMessage string
	inputBuffer   string
	isEditing     bool
}

func NewSettingsUI(screen *tcell.Screen, cfg *config.Config) *SettingsUI {
	tempConfig := &config.Config{
		BaseURL:      cfg.BaseURL,
		Username:     cfg.Username,
		Password:     cfg.Password,
		MoviesPath:   cfg.MoviesPath,
		TVShowsPath:  cfg.TVShowsPath,
		OtherPath:    cfg.OtherPath,
		ScriptFormat: cfg.ScriptFormat,
		DebugMode:    cfg.DebugMode,
	}

	return &SettingsUI{
		screen:       screen,
		config:       cfg,
		tempConfig:   tempConfig,
		currentField: FieldBaseURL,
	}
}

func (ui *SettingsUI) HandleKeyEvent(ev *tcell.EventKey) bool {
	if ui.isEditing {
		return ui.handleEditingInput(ev)
	}

	switch ev.Key() {
	case tcell.KeyUp:
		ui.moveCursorUp()
	case tcell.KeyDown:
		ui.moveCursorDown()
	case tcell.KeyEnter:
		return ui.handleEnter()
	case tcell.KeyEscape:
		return true
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'k':
			ui.moveCursorUp()
		case 'j':
			ui.moveCursorDown()
		case 'q':
			return true
		}
	}

	return false
}

func (ui *SettingsUI) handleEditingInput(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEnter:
		ui.finishEditing()
		return false
	case tcell.KeyEscape:
		ui.cancelEditing()
		return false
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(ui.inputBuffer) > 0 {
			ui.inputBuffer = ui.inputBuffer[:len(ui.inputBuffer)-1]
		}
	case tcell.KeyRune:
		ui.inputBuffer += string(ev.Rune())
	}
	return false
}

func (ui *SettingsUI) moveCursorUp() {
	if ui.currentField > 0 {
		ui.currentField--
	}
}

func (ui *SettingsUI) moveCursorDown() {
	if ui.currentField < FieldCancel {
		ui.currentField++
	}
}

func (ui *SettingsUI) handleEnter() bool {
	switch ui.currentField {
	case FieldBaseURL:
		ui.startEditing(ui.tempConfig.BaseURL)
	case FieldUsername:
		ui.startEditing(ui.tempConfig.Username)
	case FieldPassword:
		ui.startEditing(ui.tempConfig.Password)
	case FieldMoviesPath:
		ui.browsePath("Movies Path", &ui.tempConfig.MoviesPath)
	case FieldTVShowsPath:
		ui.browsePath("TV Shows Path", &ui.tempConfig.TVShowsPath)
	case FieldOtherPath:
		ui.browsePath("Other Path", &ui.tempConfig.OtherPath)
	case FieldScriptFormat:
		ui.cycleScriptFormat()
	case FieldDebugMode:
		ui.toggleDebugMode()
	case FieldSave:
		return ui.saveSettings()
	case FieldCancel:
		return true
	}
	return false
}

func (ui *SettingsUI) startEditing(currentValue string) {
	ui.isEditing = true
	ui.inputBuffer = currentValue
	ui.statusMessage = "Press Enter to save, Esc to cancel"
}

func (ui *SettingsUI) finishEditing() {
	ui.isEditing = false

	switch ui.currentField {
	case FieldBaseURL:
		ui.tempConfig.BaseURL = ui.inputBuffer
	case FieldUsername:
		ui.tempConfig.Username = ui.inputBuffer
	case FieldPassword:
		ui.tempConfig.Password = ui.inputBuffer
	case FieldMoviesPath:
		ui.tempConfig.MoviesPath = ui.inputBuffer
	case FieldTVShowsPath:
		ui.tempConfig.TVShowsPath = ui.inputBuffer
	case FieldOtherPath:
		ui.tempConfig.OtherPath = ui.inputBuffer
	}

	ui.inputBuffer = ""
	ui.statusMessage = "Press Enter to edit field, arrows to navigate"
}

func (ui *SettingsUI) cancelEditing() {
	ui.isEditing = false
	ui.inputBuffer = ""
	ui.statusMessage = "Edit cancelled"
}

func (ui *SettingsUI) cycleScriptFormat() {
	switch ui.tempConfig.ScriptFormat {
	case config.ScriptFormatBAT:
		ui.tempConfig.ScriptFormat = config.ScriptFormatSH
		ui.statusMessage = "Script format: sh"
	case config.ScriptFormatSH:
		ui.tempConfig.ScriptFormat = config.ScriptFormatTXT
		ui.statusMessage = "Script format: txt"
	case config.ScriptFormatTXT:
		ui.tempConfig.ScriptFormat = config.ScriptFormatBAT
		ui.statusMessage = "Script format: bat"
	default:
		ui.tempConfig.ScriptFormat = config.ScriptFormatBAT
		ui.statusMessage = "Script format: bat"
	}
}

func (ui *SettingsUI) toggleDebugMode() {
	ui.tempConfig.DebugMode = !ui.tempConfig.DebugMode
	if ui.tempConfig.DebugMode {
		ui.statusMessage = "Debug mode enabled - restart app to take effect"
	} else {
		ui.statusMessage = "Debug mode disabled - restart app to take effect"
	}
}

func (ui *SettingsUI) browsePath(title string, pathField *string) {
	browser := NewDirectoryBrowserUI(ui.screen, *pathField, title)

	for {
		browser.Draw()

		ev := (*ui.screen).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if result := browser.HandleKeyEvent(ev); result != nil {
				// User selected a path
				*pathField = *result
				ui.statusMessage = fmt.Sprintf("Path updated: %s", *result)
				return
			} else if ev.Key() == tcell.KeyEscape || (ev.Key() == tcell.KeyRune && ev.Rune() == 'q') {
				// User cancelled
				ui.statusMessage = "Path selection cancelled"
				return
			}
		case *tcell.EventResize:
			(*ui.screen).Sync()
		}
	}
}

func (ui *SettingsUI) saveSettings() bool {
	if err := ui.tempConfig.Validate(); err != nil {
		ui.statusMessage = fmt.Sprintf("Validation error: %v", err)
		return false
	}

	ui.config.BaseURL = ui.tempConfig.BaseURL
	ui.config.Username = ui.tempConfig.Username
	ui.config.Password = ui.tempConfig.Password
	ui.config.MoviesPath = ui.tempConfig.MoviesPath
	ui.config.TVShowsPath = ui.tempConfig.TVShowsPath
	ui.config.OtherPath = ui.tempConfig.OtherPath
	ui.config.ScriptFormat = ui.tempConfig.ScriptFormat
	ui.config.DebugMode = ui.tempConfig.DebugMode

	if err := ui.config.Save(); err != nil {
		ui.statusMessage = fmt.Sprintf("Save error: %v", err)
		return false
	}

	ui.statusMessage = "Settings saved successfully!"
	return true
}

func (ui *SettingsUI) Draw() {
	screen := *ui.screen
	w, h := screen.Size()
	screen.Clear()

	ui.drawHeader(w)
	ui.drawFields(w, h)
	ui.drawStatusBar(w, h)

	screen.Show()
}

func (ui *SettingsUI) drawHeader(w int) {
	title := "PlexLinker - Settings"
	ui.drawText(0, 0, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite), title)

	subtitle := "Configure connection and output settings"
	ui.drawText(0, 1, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), subtitle)
}

func (ui *SettingsUI) drawFields(w, h int) {
	startY := 3

	fields := []struct {
		label string
		value string
		field SettingsField
	}{
		{"Base URL:", ui.getDisplayValue(ui.tempConfig.BaseURL), FieldBaseURL},
		{"Username:", ui.getDisplayValue(ui.tempConfig.Username), FieldUsername},
		{"Password:", ui.maskPassword(ui.tempConfig.Password), FieldPassword},
		{"Movies Path:", ui.getDisplayValue(ui.tempConfig.MoviesPath), FieldMoviesPath},
		{"TV Shows Path:", ui.getDisplayValue(ui.tempConfig.TVShowsPath), FieldTVShowsPath},
		{"Other Path:", ui.getDisplayValue(ui.tempConfig.OtherPath), FieldOtherPath},
		{"Script Format:", ui.getScriptFormatDisplay(), FieldScriptFormat},
		{"Debug Mode:", ui.getDebugModeDisplay(), FieldDebugMode},
	}

	for i, field := range fields {
		y := startY + i*2
		style := tcell.StyleDefault

		if field.field == ui.currentField {
			style = style.Background(tcell.ColorNavy)
		}

		ui.drawText(2, y, style.Foreground(tcell.ColorWhite), field.label)

		valueText := field.value
		if ui.isEditing && field.field == ui.currentField {
			valueText = ui.inputBuffer + "_"
		}

		// Don't show input buffer cursor for toggle fields
		if field.field == FieldDebugMode || field.field == FieldScriptFormat {
			valueText = field.value
		}

		maxValueWidth := w - len(field.label) - 4
		if len(valueText) > maxValueWidth {
			valueText = valueText[:maxValueWidth-3] + "..."
		}

		valueStyle := style.Foreground(tcell.ColorYellow)
		if ui.isEditing && field.field == ui.currentField {
			valueStyle = valueStyle.Foreground(tcell.ColorGreen)
		}

		ui.drawText(2+len(field.label)+1, y, valueStyle, valueText)
	}

	buttonY := startY + len(fields)*2 + 1

	ui.drawButton(2, buttonY, "Save", FieldSave)
	ui.drawButton(12, buttonY, "Cancel", FieldCancel)
}

func (ui *SettingsUI) drawButton(x, y int, text string, field SettingsField) {
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite)

	if field == ui.currentField {
		style = style.Background(tcell.ColorNavy).Bold(true)
		text = fmt.Sprintf("[ %s ]", text)
	} else {
		text = fmt.Sprintf("  %s  ", text)
	}

	ui.drawText(x, y, style, text)
}

func (ui *SettingsUI) drawStatusBar(w, h int) {
	statusY := h - 2
	commandsY := h - 1

	statusText := ui.statusMessage
	if statusText == "" {
		if ui.isEditing {
			statusText = "Press Enter to save, Esc to cancel"
		} else {
			statusText = "Press Enter to edit field, arrows to navigate"
		}
	}

	ui.drawText(0, statusY, tcell.StyleDefault.Foreground(tcell.ColorYellow), statusText)

	commands := "Enter:Edit/Save  Arrows:Navigate  Esc:Cancel/Exit"
	ui.drawText(0, commandsY, tcell.StyleDefault.Foreground(tcell.ColorWhite), commands)
}

func (ui *SettingsUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *ui.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}

func (ui *SettingsUI) getDisplayValue(value string) string {
	if value == "" {
		return "<not set>"
	}
	return value
}

func (ui *SettingsUI) maskPassword(password string) string {
	if password == "" {
		return "<not set>"
	}
	return strings.Repeat("*", len(password))
}

func (ui *SettingsUI) getScriptFormatDisplay() string {
	return fmt.Sprintf("[%s] (Press Enter to cycle)", ui.tempConfig.ScriptFormat)
}

func (ui *SettingsUI) getDebugModeDisplay() string {
	if ui.tempConfig.DebugMode {
		return "[✓] Enabled"
	}
	return "[ ] Disabled"
}
