package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type ScriptViewerUI struct {
	screen       *tcell.Screen
	scriptPath   string
	scriptLines  []string
	scrollOffset int
	title        string
}

func NewScriptViewerUI(screen *tcell.Screen, scriptPath, scriptContent string) *ScriptViewerUI {
	lines := strings.Split(scriptContent, "\n")

	// Remove empty lines at the end
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	return &ScriptViewerUI{
		screen:       screen,
		scriptPath:   scriptPath,
		scriptLines:  lines,
		scrollOffset: 0,
		title:        "aria2c Download Script",
	}
}

func (ui *ScriptViewerUI) HandleKeyEvent(ev *tcell.EventKey) bool {
	_, h := (*ui.screen).Size()
	maxVisibleLines := h - 6 // Account for header and footer space

	switch ev.Key() {
	case tcell.KeyUp:
		ui.scrollUp()
	case tcell.KeyDown:
		ui.scrollDown(maxVisibleLines)
	case tcell.KeyPgUp:
		ui.scrollPageUp(maxVisibleLines)
	case tcell.KeyPgDn:
		ui.scrollPageDown(maxVisibleLines)
	case tcell.KeyHome:
		ui.scrollOffset = 0
	case tcell.KeyEnd:
		ui.scrollToEnd(maxVisibleLines)
	case tcell.KeyEscape:
		return true
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'k':
			ui.scrollUp()
		case 'j':
			ui.scrollDown(maxVisibleLines)
		case 'q':
			return true
		case 'g':
			ui.scrollOffset = 0
		case 'G':
			ui.scrollToEnd(maxVisibleLines)
		}
	}

	return false
}

func (ui *ScriptViewerUI) scrollUp() {
	if ui.scrollOffset > 0 {
		ui.scrollOffset--
	}
}

func (ui *ScriptViewerUI) scrollDown(maxVisibleLines int) {
	maxScroll := len(ui.scriptLines) - maxVisibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}

	if ui.scrollOffset < maxScroll {
		ui.scrollOffset++
	}
}

func (ui *ScriptViewerUI) scrollPageUp(maxVisibleLines int) {
	ui.scrollOffset -= maxVisibleLines / 2
	if ui.scrollOffset < 0 {
		ui.scrollOffset = 0
	}
}

func (ui *ScriptViewerUI) scrollPageDown(maxVisibleLines int) {
	maxScroll := len(ui.scriptLines) - maxVisibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}

	ui.scrollOffset += maxVisibleLines / 2
	if ui.scrollOffset > maxScroll {
		ui.scrollOffset = maxScroll
	}
}

func (ui *ScriptViewerUI) scrollToEnd(maxVisibleLines int) {
	maxScroll := len(ui.scriptLines) - maxVisibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	ui.scrollOffset = maxScroll
}

func (ui *ScriptViewerUI) Draw() {
	screen := *ui.screen
	w, h := screen.Size()
	screen.Clear()

	ui.drawHeader(w)
	ui.drawScriptContent(w, h)
	ui.drawStatusBar(w, h)

	screen.Show()
}

func (ui *ScriptViewerUI) drawHeader(w int) {
	ui.drawText(0, 0, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite), ui.title)

	pathText := fmt.Sprintf("File: %s", ui.scriptPath)
	if len(pathText) > w {
		pathText = "..." + pathText[len(pathText)-w+3:]
	}
	ui.drawText(0, 1, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), pathText)

	lineCount := fmt.Sprintf("Lines: %d", len(ui.scriptLines))
	ui.drawText(w-len(lineCount), 1, tcell.StyleDefault.Foreground(tcell.ColorGreen), lineCount)
}

func (ui *ScriptViewerUI) drawScriptContent(w, h int) {
	startY := 3
	maxVisibleLines := h - 6

	for i := 0; i < maxVisibleLines && i+ui.scrollOffset < len(ui.scriptLines); i++ {
		lineIndex := i + ui.scrollOffset
		line := ui.scriptLines[lineIndex]
		y := startY + i

		// Truncate long lines
		if len(line) > w-6 {
			line = line[:w-9] + "..."
		}

		// Line number
		lineNumText := fmt.Sprintf("%3d: ", lineIndex+1)
		ui.drawText(0, y, tcell.StyleDefault.Foreground(tcell.ColorWhite), lineNumText)

		// Line content with syntax highlighting
		style := tcell.StyleDefault.Foreground(tcell.ColorLightGray)
		if strings.HasPrefix(strings.TrimSpace(line), "aria2c") {
			style = tcell.StyleDefault.Foreground(tcell.ColorLightGreen)
		} else if strings.HasPrefix(strings.TrimSpace(line), "#") {
			style = tcell.StyleDefault.Foreground(tcell.ColorBlue)
		} else if strings.HasPrefix(strings.TrimSpace(line), "echo") {
			style = tcell.StyleDefault.Foreground(tcell.ColorYellow)
		}

		ui.drawText(5, y, style, line)
	}

	// Draw scroll indicator if needed
	if len(ui.scriptLines) > maxVisibleLines {
		scrollInfo := fmt.Sprintf("(%d-%d of %d)",
			ui.scrollOffset+1,
			ui.scrollOffset+maxVisibleLines,
			len(ui.scriptLines))
		if ui.scrollOffset+maxVisibleLines > len(ui.scriptLines) {
			scrollInfo = fmt.Sprintf("(%d-%d of %d)",
				ui.scrollOffset+1,
				len(ui.scriptLines),
				len(ui.scriptLines))
		}

		ui.drawText(w-len(scrollInfo), startY-1, tcell.StyleDefault.Foreground(tcell.ColorLightCyan), scrollInfo)
	}
}

func (ui *ScriptViewerUI) drawStatusBar(w, h int) {
	statusY := h - 2
	commandsY := h - 1

	statusText := "Script generated and copied to clipboard! Review the commands below."
	if len(statusText) > w {
		statusText = statusText[:w-3] + "..."
	}
	ui.drawText(0, statusY, tcell.StyleDefault.Foreground(tcell.ColorGreen), statusText)

	commands := "↑↓/jk:Scroll  PgUp/PgDn:Page  Home/g:Top  End/G:Bottom  Esc/q:Exit"
	if len(commands) > w {
		commands = "↑↓:Scroll  PgUp/PgDn:Page  Home:Top  End:Bottom  Esc:Exit"
	}
	ui.drawText(0, commandsY, tcell.StyleDefault.Foreground(tcell.ColorWhite), commands)
}

func (ui *ScriptViewerUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *ui.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}