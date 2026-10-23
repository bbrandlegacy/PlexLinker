package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type ShowSelectorUI struct {
	screen       *tcell.Screen
	shows        []string
	cursorIndex  int
	scrollOffset int
}

func NewShowSelectorUI(screen *tcell.Screen) *ShowSelectorUI {
	return &ShowSelectorUI{
		screen:       screen,
		shows:        []string{},
		cursorIndex:  0,
		scrollOffset: 0,
	}
}

// LoadShowsFromDirectory scans W:\shows for existing folders
func (s *ShowSelectorUI) LoadShowsFromDirectory(basePath string) error {
	// Read directory contents
	files, err := os.ReadDir(basePath)
	if err != nil {
		return fmt.Errorf("failed to read shows directory: %v", err)
	}

	s.shows = []string{}
	for _, file := range files {
		if file.IsDir() {
			s.shows = append(s.shows, file.Name())
		}
	}

	// Sort alphabetically
	sort.Strings(s.shows)

	if len(s.shows) == 0 {
		return fmt.Errorf("no show folders found in %s", basePath)
	}

	return nil
}

// SetManualShows allows manually setting show list (for testing or remote scenarios)
func (s *ShowSelectorUI) SetManualShows(shows []string) {
	s.shows = shows
	sort.Strings(s.shows)
}

func (s *ShowSelectorUI) HandleKeyEvent(ev *tcell.EventKey) *string {
	switch ev.Key() {
	case tcell.KeyEscape:
		return nil // User cancelled
	case tcell.KeyUp:
		s.moveCursorUp()
	case tcell.KeyDown:
		s.moveCursorDown()
	case tcell.KeyEnter:
		if s.cursorIndex >= 0 && s.cursorIndex < len(s.shows) {
			result := s.shows[s.cursorIndex]
			return &result
		}
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'k':
			s.moveCursorUp()
		case 'j':
			s.moveCursorDown()
		}
	}
	return nil
}

func (s *ShowSelectorUI) moveCursorUp() {
	if s.cursorIndex > 0 {
		s.cursorIndex--
		s.adjustScroll()
	}
}

func (s *ShowSelectorUI) moveCursorDown() {
	if s.cursorIndex < len(s.shows)-1 {
		s.cursorIndex++
		s.adjustScroll()
	}
}

func (s *ShowSelectorUI) adjustScroll() {
	maxVisible := 15 // Max items visible in modal
	if s.cursorIndex < s.scrollOffset {
		s.scrollOffset = s.cursorIndex
	}
	if s.cursorIndex >= s.scrollOffset+maxVisible {
		s.scrollOffset = s.cursorIndex - maxVisible + 1
	}
}

func (s *ShowSelectorUI) Draw() {
	screen := *s.screen
	w, h := screen.Size()

	// Modal dimensions
	modalWidth := 60
	modalHeight := 20
	startX := (w - modalWidth) / 2
	startY := (h - modalHeight) / 2

	// Draw modal background
	for y := startY; y < startY+modalHeight; y++ {
		for x := startX; x < startX+modalWidth; x++ {
			screen.SetContent(x, y, ' ', nil, tcell.StyleDefault.Background(tcell.ColorDarkBlue))
		}
	}

	// Draw border
	s.drawBorder(startX, startY, modalWidth, modalHeight)

	// Draw title
	title := "Select TV Show Folder"
	titleX := startX + (modalWidth-len(title))/2
	s.drawText(titleX, startY+1, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite).Background(tcell.ColorDarkBlue), title)

	// Draw shows list
	listStartY := startY + 3
	maxVisible := 15
	endIdx := s.scrollOffset + maxVisible
	if endIdx > len(s.shows) {
		endIdx = len(s.shows)
	}

	for i := s.scrollOffset; i < endIdx; i++ {
		show := s.shows[i]
		y := listStartY + (i - s.scrollOffset)

		style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDarkBlue)
		if i == s.cursorIndex {
			style = style.Background(tcell.ColorNavy).Bold(true)
		}

		// Truncate if too long
		displayText := show
		maxLen := modalWidth - 6
		if len(displayText) > maxLen {
			displayText = displayText[:maxLen-3] + "..."
		}

		s.drawText(startX+3, y, style, displayText)
	}

	// Draw scroll indicator if needed
	if len(s.shows) > maxVisible {
		scrollInfo := fmt.Sprintf("(%d-%d of %d)", s.scrollOffset+1, endIdx, len(s.shows))
		scrollX := startX + modalWidth - len(scrollInfo) - 2
		s.drawText(scrollX, startY+2, tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorDarkBlue), scrollInfo)
	}

	// Draw instructions
	instructions := "↑↓ or j/k: navigate | Enter: select | Esc: cancel"
	instX := startX + (modalWidth-len(instructions))/2
	s.drawText(instX, startY+modalHeight-2, tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorDarkBlue), instructions)

	screen.Show()
}

func (s *ShowSelectorUI) drawBorder(x, y, w, h int) {
	screen := *s.screen
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDarkBlue)

	// Top and bottom borders
	for i := x; i < x+w; i++ {
		screen.SetContent(i, y, '─', nil, style)
		screen.SetContent(i, y+h-1, '─', nil, style)
	}

	// Left and right borders
	for i := y; i < y+h; i++ {
		screen.SetContent(x, i, '│', nil, style)
		screen.SetContent(x+w-1, i, '│', nil, style)
	}

	// Corners
	screen.SetContent(x, y, '┌', nil, style)
	screen.SetContent(x+w-1, y, '┐', nil, style)
	screen.SetContent(x, y+h-1, '└', nil, style)
	screen.SetContent(x+w-1, y+h-1, '┘', nil, style)
}

func (s *ShowSelectorUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *s.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}

// ParseShowNameFromURL attempts to extract a show name from the URL path
func ParseShowNameFromURL(url string) string {
	// Remove protocol and host
	parts := strings.Split(url, "/")

	// Look for likely show name in path (usually after "shows" or similar)
	for i, part := range parts {
		if strings.ToLower(part) == "shows" || strings.ToLower(part) == "tv" {
			if i+1 < len(parts) {
				return filepath.Base(parts[i+1])
			}
		}
	}

	// Fallback: return the last meaningful directory name
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" && !strings.Contains(parts[i], ".") {
			return filepath.Base(parts[i])
		}
	}

	return ""
}
