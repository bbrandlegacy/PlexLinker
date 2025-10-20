package ui

import (
	"github.com/gdamore/tcell/v2"
)

type ConfirmModalUI struct {
	screen      *tcell.Screen
	message     string
	confirmText string
	cancelText  string
	selected    bool // true = confirm, false = cancel
}

func NewConfirmModalUI(screen *tcell.Screen, message string) *ConfirmModalUI {
	return &ConfirmModalUI{
		screen:      screen,
		message:     message,
		confirmText: "Yes",
		cancelText:  "No",
		selected:    true, // Default to "Yes" (confirm)
	}
}

func (ui *ConfirmModalUI) HandleKeyEvent(ev *tcell.EventKey) *bool {
	switch ev.Key() {
	case tcell.KeyLeft, tcell.KeyRight, tcell.KeyTab:
		ui.selected = !ui.selected
		return nil
	case tcell.KeyEnter:
		result := ui.selected
		return &result
	case tcell.KeyEscape:
		result := false
		return &result
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'h':
			ui.selected = true
			return nil
		case 'l':
			ui.selected = false
			return nil
		case 'y', 'Y':
			result := true
			return &result
		case 'n', 'N':
			result := false
			return &result
		}
	}
	return nil
}

func (ui *ConfirmModalUI) Draw() {
	screen := *ui.screen
	w, h := screen.Size()

	// Calculate modal dimensions
	modalWidth := 60
	modalHeight := 7
	if len(ui.message)+4 > modalWidth {
		modalWidth = len(ui.message) + 4
	}
	if w < modalWidth+4 {
		modalWidth = w - 4
	}

	startX := (w - modalWidth) / 2
	startY := (h - modalHeight) / 2

	// Draw modal background
	bgStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	for y := startY; y < startY+modalHeight; y++ {
		for x := startX; x < startX+modalWidth; x++ {
			screen.SetContent(x, y, ' ', nil, bgStyle)
		}
	}

	// Draw border
	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	for x := startX; x < startX+modalWidth; x++ {
		screen.SetContent(x, startY, '─', nil, borderStyle)
		screen.SetContent(x, startY+modalHeight-1, '─', nil, borderStyle)
	}
	for y := startY; y < startY+modalHeight; y++ {
		screen.SetContent(startX, y, '│', nil, borderStyle)
		screen.SetContent(startX+modalWidth-1, y, '│', nil, borderStyle)
	}
	screen.SetContent(startX, startY, '┌', nil, borderStyle)
	screen.SetContent(startX+modalWidth-1, startY, '┐', nil, borderStyle)
	screen.SetContent(startX, startY+modalHeight-1, '└', nil, borderStyle)
	screen.SetContent(startX+modalWidth-1, startY+modalHeight-1, '┘', nil, borderStyle)

	// Draw message
	messageY := startY + 2
	messageX := startX + (modalWidth-len(ui.message))/2
	if messageX < startX+2 {
		messageX = startX + 2
	}
	ui.drawText(messageX, messageY, bgStyle, ui.message)

	// Draw buttons
	buttonY := startY + 4
	buttonSpacing := 10
	totalButtonWidth := len(ui.confirmText) + len(ui.cancelText) + buttonSpacing
	buttonsStartX := startX + (modalWidth-totalButtonWidth)/2

	// Confirm button
	confirmStyle := bgStyle
	confirmText := "  " + ui.confirmText + "  "
	if ui.selected {
		confirmStyle = tcell.StyleDefault.Background(tcell.ColorGreen).Foreground(tcell.ColorBlack).Bold(true)
		confirmText = "[ " + ui.confirmText + " ]"
	}
	ui.drawText(buttonsStartX, buttonY, confirmStyle, confirmText)

	// Cancel button
	cancelX := buttonsStartX + len(confirmText) + buttonSpacing
	cancelStyle := bgStyle
	cancelText := "  " + ui.cancelText + "  "
	if !ui.selected {
		cancelStyle = tcell.StyleDefault.Background(tcell.ColorRed).Foreground(tcell.ColorBlack).Bold(true)
		cancelText = "[ " + ui.cancelText + " ]"
	}
	ui.drawText(cancelX, buttonY, cancelStyle, cancelText)

	screen.Show()
}

func (ui *ConfirmModalUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *ui.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}