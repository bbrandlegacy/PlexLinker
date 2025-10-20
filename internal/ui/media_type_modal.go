package ui

import (
	"github.com/gdamore/tcell/v2"
)

type MediaType string

const (
	MediaTypeMovies  MediaType = "movies"
	MediaTypeTVShows MediaType = "shows"
)

type MediaTypeModalUI struct {
	screen        *tcell.Screen
	selectedIndex int // 0 = Movies, 1 = TV Shows
}

func NewMediaTypeModalUI(screen *tcell.Screen) *MediaTypeModalUI {
	return &MediaTypeModalUI{
		screen:        screen,
		selectedIndex: 0,
	}
}

func (m *MediaTypeModalUI) HandleKeyEvent(ev *tcell.EventKey) *MediaType {
	switch ev.Key() {
	case tcell.KeyEscape:
		return nil // User cancelled
	case tcell.KeyLeft:
		m.selectedIndex = 0
	case tcell.KeyRight:
		m.selectedIndex = 1
	case tcell.KeyEnter:
		if m.selectedIndex == 0 {
			result := MediaTypeMovies
			return &result
		}
		result := MediaTypeTVShows
		return &result
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'h':
			m.selectedIndex = 0
		case 'l':
			m.selectedIndex = 1
		}
	}
	return nil
}

func (m *MediaTypeModalUI) Draw() {
	screen := *m.screen
	w, h := screen.Size()

	// Modal dimensions
	modalWidth := 50
	modalHeight := 8
	startX := (w - modalWidth) / 2
	startY := (h - modalHeight) / 2

	// Draw modal background
	for y := startY; y < startY+modalHeight; y++ {
		for x := startX; x < startX+modalWidth; x++ {
			screen.SetContent(x, y, ' ', nil, tcell.StyleDefault.Background(tcell.ColorDarkBlue))
		}
	}

	// Draw border
	m.drawBorder(startX, startY, modalWidth, modalHeight)

	// Draw title
	title := "Select Media Type"
	titleX := startX + (modalWidth-len(title))/2
	m.drawText(titleX, startY+1, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite).Background(tcell.ColorDarkBlue), title)

	// Draw options
	moviesText := "Movies (W:\\movies)"
	tvShowsText := "TV Shows (W:\\shows)"

	moviesY := startY + 3
	tvShowsY := startY + 4

	// Draw Movies option
	moviesStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDarkBlue)
	if m.selectedIndex == 0 {
		moviesStyle = moviesStyle.Background(tcell.ColorNavy).Bold(true)
	}
	moviesX := startX + (modalWidth-len(moviesText))/2
	m.drawText(moviesX, moviesY, moviesStyle, moviesText)

	// Draw TV Shows option
	tvShowsStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDarkBlue)
	if m.selectedIndex == 1 {
		tvShowsStyle = tvShowsStyle.Background(tcell.ColorNavy).Bold(true)
	}
	tvShowsX := startX + (modalWidth-len(tvShowsText))/2
	m.drawText(tvShowsX, tvShowsY, tvShowsStyle, tvShowsText)

	// Draw instructions
	instructions := "← → or h/l to select, Enter to confirm, Esc to cancel"
	instX := startX + (modalWidth-len(instructions))/2
	m.drawText(instX, startY+modalHeight-2, tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorDarkBlue), instructions)

	screen.Show()
}

func (m *MediaTypeModalUI) drawBorder(x, y, w, h int) {
	screen := *m.screen
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

func (m *MediaTypeModalUI) drawText(x, y int, style tcell.Style, text string) {
	screen := *m.screen
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}
