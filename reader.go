package main

import (
	"bufio"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/text/width"
	"io"
	"log"
	"math"
	"os"
	"time"
	"unicode"
	"unicode/utf8"
)

// TODO global variables?
var paused bool = true
var wordsPerMinute int = 300
var displayedWord string = ""
var singleCharacter bool = false
var currentByteIndex int = 0
var maxByteIndex int = 0

var spinner []string = []string{"⠁", "⠈", "⠐", "⠂"}
var spinnerIndex int = 0

func spinnerInc() {
	spinnerIndex = (spinnerIndex + 1) % len(spinner)
}

func runeWidth(r rune) int {
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	default:
		return 1
	}
}

func write(s tcell.Screen, word string, col int, row int) {
	i := 0
	for _, c := range word {
		s.SetContent(col+i, row, c, nil, tcell.StyleDefault)
		i += runeWidth(c)
	}
	s.Show()
}

func writeWord(s tcell.Screen, word string) {
	width, height := s.Size()
	write(
		s,
		word,
		width/2-utf8.RuneCountInString(word)/2,
		height/2,
	)
}

func statusHelp() string {
	if paused {
		return "[Press SPC to start.]"
	} else {
		return ""
	}
}

func statusProgress() string {
	// width: 16 * 2 = 32
	const runeAmount int = 16
	const width int = runeAmount * 2
	completed := int(math.Round(float64(currentPos) / float64(maxPos) * float64(width)))

	s := ""
	double := completed / 2
	for i := 0; i < double; i++ {
		s += "⠿"
	}

	if double*2 < completed {
		s += "⠇"
	}

	for utf8.RuneCountInString(s) < runeAmount {
		s += " "
	}

	return s
}

func writeStatus(s tcell.Screen, word string) {
	width, height := s.Size()
	write(s, spinner[spinnerIndex], 0, height-1)

	help := statusHelp()
	write(s, help, width/2-utf8.RuneCountInString(help)/2, height-2)

	progress := statusProgress()
	write(s, progress, width/2-utf8.RuneCountInString(progress)/2, height-1)

	write(s, word, width-utf8.RuneCountInString(word), height-1)
}

func updateUI(s tcell.Screen) {
	s.Clear()

	unit := "words"
	if singleCharacter {
		unit = "characters"
	}
	writeStatus(s, fmt.Sprintf("%d %s per min", wordsPerMinute, unit))

	writeWord(s, displayedWord)
}

func handleComms(comm chan int) bool {
	const speedInc = 5
	handledMessage := false
	messagesPending := true
	for messagesPending {
		select {
		case msg := <-comm:
			switch msg {
			case COMM_SPEED_INC:
				wordsPerMinute += speedInc
			case COMM_SPEED_DEC:
				wordsPerMinute -= speedInc
			case COMM_TOGGLE:
				paused = !paused
			case COMM_SINGLE_CHARACTER:
				singleCharacter = !singleCharacter
			}
			handledMessage = true
		default:
			messagesPending = false
		}
	}

	return handledMessage
}

func getDelayMs() int64 {
	return int64(math.Round(1_000 / (float64(wordsPerMinute) / float64(60))))
}

// Waits but still handles comms at 60 Hz
func wait(s tcell.Screen, comm chan int) {
	const Hz = 60
	remainingMs := getDelayMs()

	for remainingMs > 0 {
		prevTime := time.Now().UnixMilli()

		if handleComms(comm) {
			updateUI(s)
		}

		time.Sleep(1_000 / Hz * time.Millisecond)

		// Immediately exit this wait when unpausing
		if paused {
			prevTime = 0
		} else {
			remainingMs -= time.Now().UnixMilli() - prevTime
		}
	}
}

func guessSingleCharacter(r rune) bool {
	return runeWidth(r) == 2
}

func wordBoundary(singleCharacter bool, r rune) bool {
	return singleCharacter || unicode.IsSpace(r)
}

func speedRead(s tcell.Screen, text string, comm chan int) {
	word := ""

	maxByteIndex = len(text)
	rune, _ := utf8.DecodeRuneInString(text[:4])
	singleCharacter = guessSingleCharacter(rune)

	for byteIndex, rune := range text {
		currentByteIndex = byteIndex
		if word != "" && wordBoundary(singleCharacter, rune) {
			displayedWord = word
			word = ""
			spinnerInc()
			updateUI(s)
			wait(s, comm)
		}

		if !unicode.IsSpace(rune) {
			word = word + string(rune)
		}
	}
}

func mainReader(s tcell.Screen, comm chan int) {
	reader := bufio.NewReader(os.Stdin)
	buf, err := io.ReadAll(reader)
	if err != nil {
		log.Fatalf("Could not read stdin: %v\n", err)
	}

	text := string(buf)
	speedRead(s, text, comm)
}
