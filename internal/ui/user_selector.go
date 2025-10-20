package ui

import (
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

type UserSelectorMode int

const (
	ModeSelection UserSelectorMode = iota
	ModeNewUser
)

type UserSelectorUI struct {
	screen        *tcell.Screen
	usernames     []string
	selectedIndex int
	mode          UserSelectorMode
	input         string
	statusMessage string
	lastUser      string
}

func NewUserSelectorUI(screen *tcell.Screen, usernames []string, lastUser string) *UserSelectorUI {
	// Find index of last user for auto-selection
	selectedIndex := 0
	mode := ModeSelection

	// If no usernames exist, go directly to new user mode
	if len(usernames) == 0 {
		mode = ModeNewUser
	} else if lastUser != "" {
		for i, username := range usernames {
			if username == lastUser {
				selectedIndex = i
				break
			}
		}
	}

	return &UserSelectorUI{
		screen:        screen,
		usernames:     usernames,
		selectedIndex: selectedIndex,
		mode:          mode,
		input:         "",
		statusMessage: "",
		lastUser:      lastUser,
	}
}

func (ui *UserSelectorUI) Draw() {
	screen := *ui.screen
	screen.Clear()
	w, h := screen.Size()

	if ui.mode == ModeSelection {
		ui.drawSelectionMode(w, h)
	} else {
		ui.drawNewUserMode(w, h)
	}

	screen.Show()
}

func (ui *UserSelectorUI) drawSelectionMode(w, h int) {
	// Header
	headerStyle := tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorWhite).Bold(true)
	headerText := " Select User "
	ui.drawTextCentered(0, w, headerStyle, headerText)

	// Instructions
	y := 2
	defaultStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	ui.drawText(2, y, defaultStyle, "Use ↑↓ arrows to navigate, Enter to select")
	y += 2

	// User list
	for i, username := range ui.usernames {
		style := defaultStyle
		prefix := "  "
		if i == ui.selectedIndex {
			style = tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
			prefix = "> "
		}
		ui.drawText(4, y+i, style, prefix+username)
	}

	// Add new user option
	y += len(ui.usernames) + 1
	addNewStyle := defaultStyle.Foreground(tcell.ColorGreen)
	addNewPrefix := "  "
	if ui.selectedIndex == len(ui.usernames) {
		addNewStyle = addNewStyle.Bold(true)
		addNewPrefix = "> "
	}
	ui.drawText(4, y, addNewStyle, addNewPrefix+"+ Add New User")

	// Status message
	if ui.statusMessage != "" {
		ui.drawText(2, h-3, tcell.StyleDefault.Foreground(tcell.ColorRed), ui.statusMessage)
	}

	// Footer
	footerStyle := tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorWhite)
	footerText := " [↑↓]Navigate  [Enter]Select  [Esc]Quit "
	ui.drawTextCentered(h-1, w, footerStyle, footerText)
}

func (ui *UserSelectorUI) drawNewUserMode(w, h int) {
	// Header
	headerStyle := tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorWhite).Bold(true)
	headerText := " Add New User "
	ui.drawTextCentered(0, w, headerStyle, headerText)

	// Instructions
	y := 3
	defaultStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	ui.drawText(2, y, defaultStyle, "Enter username (alphanumeric, underscore, hyphen only):")
	y += 2

	// Input field
	inputStyle := tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite)
	inputText := ui.input + "_"
	ui.drawText(4, y, inputStyle, inputText+strings.Repeat(" ", 30-len(inputText)))

	// Status message
	if ui.statusMessage != "" {
		ui.drawText(2, h-3, tcell.StyleDefault.Foreground(tcell.ColorRed), ui.statusMessage)
	}

	// Footer
	footerStyle := tcell.StyleDefault.Background(tcell.ColorNavy).Foreground(tcell.ColorWhite)
	footerText := " [Enter]Confirm  [Esc]Cancel "
	ui.drawTextCentered(h-1, w, footerStyle, footerText)
}

// HandleKeyEvent returns selected username when Enter is pressed, empty string otherwise
func (ui *UserSelectorUI) HandleKeyEvent(ev *tcell.EventKey) string {
	if ui.mode == ModeSelection {
		return ui.handleSelectionKeys(ev)
	}
	return ui.handleNewUserKeys(ev)
}

func (ui *UserSelectorUI) handleSelectionKeys(ev *tcell.EventKey) string {
	maxIndex := len(ui.usernames) // Includes "+ Add New User" option

	switch ev.Key() {
	case tcell.KeyUp:
		if ui.selectedIndex > 0 {
			ui.selectedIndex--
		}
	case tcell.KeyDown:
		if ui.selectedIndex < maxIndex {
			ui.selectedIndex++
		}
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'k': // Vim-style up
			if ui.selectedIndex > 0 {
				ui.selectedIndex--
			}
		case 'j': // Vim-style down
			if ui.selectedIndex < maxIndex {
				ui.selectedIndex++
			}
		}
	case tcell.KeyEnter:
		if ui.selectedIndex == len(ui.usernames) {
			// "Add New User" selected
			ui.mode = ModeNewUser
			ui.input = ""
			ui.statusMessage = ""
		} else {
			// User selected
			return ui.usernames[ui.selectedIndex]
		}
	case tcell.KeyEscape:
		return "QUIT"
	}

	return ""
}

func (ui *UserSelectorUI) handleNewUserKeys(ev *tcell.EventKey) string {
	switch ev.Key() {
	case tcell.KeyEnter:
		// Validate and return new username
		if ui.input == "" {
			ui.statusMessage = "Username cannot be empty"
			return ""
		}
		if !isValidUsername(ui.input) {
			ui.statusMessage = "Invalid username format"
			return ""
		}
		// Check for duplicates
		for _, existing := range ui.usernames {
			if existing == ui.input {
				ui.statusMessage = "Username already exists"
				return ""
			}
		}
		return ui.input

	case tcell.KeyEscape:
		// Cancel new user creation
		ui.mode = ModeSelection
		ui.input = ""
		ui.statusMessage = ""

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(ui.input) > 0 {
			ui.input = ui.input[:len(ui.input)-1]
		}

	case tcell.KeyRune:
		// Add character if valid
		r := ev.Rune()
		if isValidUsernameChar(r) && len(ui.input) < 30 {
			ui.input += string(r)
		}
	}

	return ""
}

func (ui *UserSelectorUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *ui.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}

func (ui *UserSelectorUI) drawTextCentered(y, width int, style tcell.Style, text string) {
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

func isValidUsername(username string) bool {
	if username == "" || len(username) > 30 {
		return false
	}
	for _, r := range username {
		if !isValidUsernameChar(r) {
			return false
		}
	}
	return true
}

func isValidUsernameChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}
