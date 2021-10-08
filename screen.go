package main

import (
	"github.com/gdamore/tcell/v2"
	"golang.org/x/text/width"
	"math"
	"unicode/utf8"
)

type screen struct {
	tcellScreen tcell.Screen
}

// TODO: global variables?
var spinner []string = []string{"⠁", "⠈", "⠐", "⠂"}
var spinnerIndex int = 0

func spinnerInc() {
	spinnerIndex = (spinnerIndex + 1) % len(spinner)
}

func (s *screen) error(err error) {
	tcellErr := tcell.NewEventError(err)
	s.tcellScreen.PostEvent(tcellErr)
}

func (s *screen) clear() {
	s.tcellScreen.Clear()
}

func (s *screen) runeWidth(c rune) int {
	switch width.LookupRune(c).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	default:
		return 1
	}
}

func (s *screen) write(word string, col int, row int) {
	i := 0
	for _, c := range word {
		s.tcellScreen.SetContent(col+i, row, c, nil, tcell.StyleDefault)
		i += s.runeWidth(c)
	}
	s.tcellScreen.Show()
}

func (s *screen) writeWord(word string) {
	width, height := s.tcellScreen.Size()
	s.write(
		word,
		width/2-utf8.RuneCountInString(word)/2,
		height/2,
	)
}

func (s *screen) statusHelp(paused bool) string {
	if paused {
		return "[Press SPC to start.]"
	} else {
		return ""
	}
}

func (s *screen) statusProgress(completed int, total int) string {
	const runeAmount int = 32
	const width int = runeAmount * 2
	completedRatio := int(math.Round(float64(completed) / float64(total) * float64(width)))

	progress := ""
	double := completedRatio / 2
	for i := 0; i < double; i++ {
		progress += "⠿"
	}

	if double*2 < completedRatio {
		progress += "⠇"
	}

	for utf8.RuneCountInString(progress) < runeAmount {
		progress += " "
	}

	return progress
}

func (s *screen) writeStatus(word string, paused bool, completed int, total int) {
	width, height := s.tcellScreen.Size()
	s.write(spinner[spinnerIndex], 0, height-1)

	help := s.statusHelp(paused)
	s.write(help, width/2-utf8.RuneCountInString(help)/2, height-2)

	progress := s.statusProgress(completed, total)
	s.write(progress, width/2-utf8.RuneCountInString(progress)/2, height-1)

	s.write(word, width-utf8.RuneCountInString(word), height-1)
}
