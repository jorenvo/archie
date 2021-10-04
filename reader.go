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
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// TODO global variables?
var text string = ""
var paused bool = true
var wordsPerMinute int = 300
var displayedWord string = ""
var singleCharacter bool = false
var currentByteIndex int = 0
var maxByteIndex int = 0
var newWordsPerMinuteBuffer int = 0

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
	const runeAmount int = 32
	const width int = runeAmount * 2
	completed := int(math.Round(float64(currentByteIndex) / float64(maxByteIndex) * float64(width)))

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

	if newWordsPerMinuteBuffer == 0 {
		writeWord(s, displayedWord)
	} else {
		writeWord(s, fmt.Sprintf("New %s per min: %d", unit, newWordsPerMinuteBuffer))
	}
}

func handleComms(comm chan int) bool {
	const speedInc = 5
	handledMessage := false
	messagesPending := true
	for messagesPending {
		select {
		// COM_RESIZE is not explicitly handled because
		// caller calls updateUI() if handledMessage == true
		case msg := <-comm:
			handledMessage = true

			// Comms that are always handled (e.g. wpm input)
			switch {
			case msg >= COMM_DIGIT_0 && msg <= COMM_DIGIT_9:
				newWordsPerMinuteBuffer =
					newWordsPerMinuteBuffer*10 + msg - COMM_DIGIT_0
				paused = true
			case msg == COMM_BACKSPACE:
				newWordsPerMinuteBuffer /= 10
			case msg == COMM_CONFIRM:
				if newWordsPerMinuteBuffer != 0 {
					wordsPerMinute = newWordsPerMinuteBuffer
					newWordsPerMinuteBuffer = 0
				}
			}

			if newWordsPerMinuteBuffer != 0 {
				break
			}

			// Comms only handled when not inputting wpm
			switch msg {
			case COMM_SPEED_INC:
				wordsPerMinute += speedInc
			case COMM_SPEED_DEC:
				wordsPerMinute -= speedInc
			case COMM_TOGGLE:
				paused = !paused
			case COMM_SINGLE_CHARACTER:
				singleCharacter = !singleCharacter
			case COMM_SENTENCE_BACKWARD:
				currentByteIndex = strings.LastIndexAny(text[:currentByteIndex], sentenceBreaks)
			}
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
			remainingMs = getDelayMs()
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

func speedRead(s tcell.Screen, comm chan int) {
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

func stripByteOrderMark(buf []byte) []byte {
	bom := [...]byte{0xef, 0xbb, 0xbf}

	for i, bomByte := range bom {
		if buf[i] != bomByte {
			return buf
		}
	}

	return buf[3:]
}

func mainReader(s tcell.Screen, comm chan int) {
	reader := bufio.NewReader(os.Stdin)
	buf, err := io.ReadAll(reader)
	if err != nil {
		log.Fatalf("Could not read stdin: %v\n", err)
	}
	buf = stripByteOrderMark(buf)

	text = string(buf)
	speedRead(s, comm)
}
