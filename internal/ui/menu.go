package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"plexlinker/internal/config"
)

type MenuOption int

const (
	MenuBrowse MenuOption = iota
	MenuTVRenamer
	MenuSettings
	MenuQuit
)

type MainMenuUI struct {
	screen        *tcell.Screen
	config        *config.Config
	currentOption MenuOption
	statusMessage string
	currentUser   string
}

func NewMainMenuUI(screen *tcell.Screen, cfg *config.Config) *MainMenuUI {
	return &MainMenuUI{
		screen:        screen,
		config:        cfg,
		currentOption: MenuBrowse,
	}
}

func (ui *MainMenuUI) SetCurrentUser(username string) {
	ui.currentUser = username
}

func (ui *MainMenuUI) HandleKeyEvent(ev *tcell.EventKey) MenuOption {
	switch ev.Key() {
	case tcell.KeyUp:
		ui.moveCursorUp()
	case tcell.KeyDown:
		ui.moveCursorDown()
	case tcell.KeyEnter:
		return ui.currentOption
	case tcell.KeyEscape:
		return MenuQuit
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'k':
			ui.moveCursorUp()
		case 'j':
			ui.moveCursorDown()
		case 'q':
			return MenuQuit
		case '1':
			return MenuBrowse
		case '2':
			return MenuTVRenamer
		case '3':
			return MenuSettings
		case '4':
			return MenuQuit
		}
	}

	return -1
}

func (ui *MainMenuUI) moveCursorUp() {
	if ui.currentOption > 0 {
		ui.currentOption--
	}
}

func (ui *MainMenuUI) moveCursorDown() {
	if ui.currentOption < MenuQuit {
		ui.currentOption++
	}
}

func (ui *MainMenuUI) SetStatusMessage(message string) {
	ui.statusMessage = message
}

func (ui *MainMenuUI) Draw() {
	screen := *ui.screen
	w, h := screen.Size()
	screen.Clear()

	ui.drawHeader(w)
	ui.drawMenu(w, h)
	ui.drawStatusBar(w, h)

	screen.Show()
}

func (ui *MainMenuUI) drawHeader(w int) {
	title := "PlexLinker"
	subtitle := "Interactive HTML Directory Browser & aria2c Script Generator"

	// Show current user if set
	userInfo := ""
	if ui.currentUser != "" {
		userInfo = fmt.Sprintf("User: %s", ui.currentUser)
	}

	titleX := (w - len(title)) / 2
	subtitleX := (w - len(subtitle)) / 2

	ui.drawText(titleX, 2, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite), title)
	ui.drawText(subtitleX, 3, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), subtitle)

	if userInfo != "" {
		userInfoX := w - len(userInfo) - 2
		ui.drawText(userInfoX, 2, tcell.StyleDefault.Foreground(tcell.ColorYellow), userInfo)
	}
}

func (ui *MainMenuUI) drawMenu(w, h int) {
	menuItems := []struct {
		text   string
		option MenuOption
	}{
		{"1. Browse Directories", MenuBrowse},
		{"2. TV Show Renamer", MenuTVRenamer},
		{"3. Settings", MenuSettings},
		{"4. Quit", MenuQuit},
	}

	startY := h/2 - len(menuItems)/2

	for i, item := range menuItems {
		y := startY + i*2
		style := tcell.StyleDefault.Foreground(tcell.ColorWhite)

		if item.option == ui.currentOption {
			style = style.Background(tcell.ColorNavy).Bold(true)
			item.text = fmt.Sprintf("▶ %s", item.text)
		} else {
			item.text = fmt.Sprintf("  %s", item.text)
		}

		x := (w - len(item.text)) / 2
		ui.drawText(x, y, style, item.text)
	}
}

func (ui *MainMenuUI) drawStatusBar(w, h int) {
	statusY := h - 3
	configY := h - 2
	commandsY := h - 1

	statusText := ui.statusMessage
	if statusText == "" {
		statusText = "Welcome to PlexLinker! Select an option to continue."
	}

	ui.drawText(0, statusY, tcell.StyleDefault.Foreground(tcell.ColorYellow), statusText)

	configStatus := ui.getConfigStatus()
	ui.drawText(0, configY, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), configStatus)

	commands := "Enter:Select  Arrows/Numbers:Navigate  q/Esc:Quit"
	ui.drawText(0, commandsY, tcell.StyleDefault.Foreground(tcell.ColorWhite), commands)
}

func (ui *MainMenuUI) getConfigStatus() string {
	if err := ui.config.Validate(); err != nil {
		return "⚠ Configuration incomplete - please visit Settings"
	}
	return "✓ Configuration OK"
}

func (ui *MainMenuUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *ui.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}
