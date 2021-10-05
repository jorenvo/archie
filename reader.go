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

type reader struct {
	text string
	paused bool
	wordsPerMinute int
	displayedWord string
	singleCharacter bool
	currentByteIndex int
	maxByteIndex int
	newWordsPerMinuteBuffer int
}

// TODO: global variables?
var spinner []string = []string{"⠁", "⠈", "⠐", "⠂"}
var spinnerIndex int = 0

func spinnerInc() {
	spinnerIndex = (spinnerIndex + 1) % len(spinner)
}

func (r *reader) runeWidth(c rune) int {
	switch width.LookupRune(c).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	default:
		return 1
	}
}

func (r *reader) write(s tcell.Screen, word string, col int, row int) {
	i := 0
	for _, c := range word {
		s.SetContent(col+i, row, c, nil, tcell.StyleDefault)
		i += r.runeWidth(c)
	}
	s.Show()
}

func (r *reader) writeWord(s tcell.Screen, word string) {
	width, height := s.Size()
	r.write(
		s,
		word,
		width/2-utf8.RuneCountInString(word)/2,
		height/2,
	)
}

func (r *reader) statusHelp() string {
	if r.paused {
		return "[Press SPC to start.]"
	} else {
		return ""
	}
}

func (r *reader) statusProgress() string {
	const runeAmount int = 32
	const width int = runeAmount * 2
	completed := int(math.Round(float64(r.currentByteIndex) / float64(r.maxByteIndex) * float64(width)))

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

func (r *reader) writeStatus(s tcell.Screen, word string) {
	width, height := s.Size()
	r.write(s, spinner[spinnerIndex], 0, height-1)

	help := r.statusHelp()
	r.write(s, help, width/2-utf8.RuneCountInString(help)/2, height-2)

	progress := r.statusProgress()
	r.write(s, progress, width/2-utf8.RuneCountInString(progress)/2, height-1)

	r.write(s, word, width-utf8.RuneCountInString(word), height-1)
}

func (r *reader) updateUI(s tcell.Screen) {
	s.Clear()

	unit := "words"
	if r.singleCharacter {
		unit = "characters"
	}
	r.writeStatus(s, fmt.Sprintf("%d %s per min", r.wordsPerMinute, unit))

	if r.newWordsPerMinuteBuffer == 0 {
		r.writeWord(s, r.displayedWord)
	} else {
		r.writeWord(s, fmt.Sprintf("New %s per min: %d", unit, r.newWordsPerMinuteBuffer))
	}
}

type skipPastCharacterParam bool

const (
	Backwards skipPastCharacterParam = true
	Forwards                         = false
)

func (r *reader) skipPastCharacter(param skipPastCharacterParam) {
	if param == Backwards {
		// Go backwards 4 bytes and go forward until we reach our original
		// pos to figure out the character right before it. We may end up
		// with an invalid encoding in the beginning but I think that
		// should be fine.
		start := r.currentByteIndex - 4
		if start < 0 {
			return
		}

		for {
			_, offset := utf8.DecodeRuneInString(r.text[start:])
			if start+offset == r.currentByteIndex {
				r.currentByteIndex = start
				return
			}

			start += offset

			if start > r.currentByteIndex {
				log.Fatalf("Went past character we started at")
			}
		}
	} else {
		_, offset := utf8.DecodeRuneInString(r.text[r.currentByteIndex:])
		r.currentByteIndex += offset
	}
}

func (r *reader) handleComms(comm chan int) bool {
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
				r.newWordsPerMinuteBuffer =
					r.newWordsPerMinuteBuffer*10 + msg - COMM_DIGIT_0
				r.paused = true
			case msg == COMM_BACKSPACE:
				r.newWordsPerMinuteBuffer /= 10
			case msg == COMM_CONFIRM:
				if r.newWordsPerMinuteBuffer != 0 {
					r.wordsPerMinute = r.newWordsPerMinuteBuffer
					r.newWordsPerMinuteBuffer = 0
				}
			}

			if r.newWordsPerMinuteBuffer != 0 {
				break
			}

			// Comms only handled when not inputting wpm
			switch msg {
			case COMM_SPEED_INC:
				r.wordsPerMinute += speedInc
			case COMM_SPEED_DEC:
				r.wordsPerMinute -= speedInc
			case COMM_TOGGLE:
				r.paused = !r.paused
			case COMM_SINGLE_CHARACTER:
				r.singleCharacter = !r.singleCharacter
			case COMM_SENTENCE_BACKWARD:
				skippedCharactersBackwards := 0
				startingByteIndex := r.currentByteIndex
				for startingByteIndex == r.currentByteIndex {
					for i := 0; i < skippedCharactersBackwards; i++ {
						r.skipPastCharacter(Backwards)
					}
					skippedCharactersBackwards++

					previousBreak := strings.LastIndexAny(r.text[:r.currentByteIndex], sentenceBreaks)
					if previousBreak == -1 {
						// No break found, go back to the beginning of file
						r.currentByteIndex = 0
						r.displayedWord = r.nextWord()
						break
					} else {
						r.currentByteIndex = previousBreak
						r.skipPastCharacter(Forwards)
						r.displayedWord = r.nextWord()
					}
				}
			case COMM_SENTENCE_FORWARD:
				// Skip this character in case it's a period
				r.skipPastCharacter(Forwards)
				nextBreak := strings.IndexAny(r.text[r.currentByteIndex:], sentenceBreaks)
				if nextBreak == -1 {
					break
				}

				// += because IndexAny ran on substring
				r.currentByteIndex += nextBreak

				r.skipPastCharacter(Forwards)
				r.displayedWord = r.nextWord()
			}
		default:
			messagesPending = false
		}
	}

	return handledMessage
}

func (r *reader) getDelayMs() int64 {
	return int64(math.Round(1_000 / (float64(r.wordsPerMinute) / float64(60))))
}

// Waits but still handles comms at 60 Hz
func (r *reader) wait(s tcell.Screen, comm chan int) {
	const Hz = 60
	remainingMs := r.getDelayMs()

	for remainingMs > 0 {
		prevTime := time.Now().UnixMilli()

		if r.handleComms(comm) {
			r.updateUI(s)
			remainingMs = r.getDelayMs()
		}

		time.Sleep(1_000 / Hz * time.Millisecond)

		// Immediately exit this wait when unpausing
		if r.paused {
			prevTime = 0
		} else {
			remainingMs -= time.Now().UnixMilli() - prevTime
		}
	}
}

func (r *reader) guessSingleCharacter(c rune) bool {
	return r.runeWidth(c) == 2
}

func (r *reader) wordBoundary(singleCharacter bool, c rune) bool {
	// TODO: IsPunct will include quotes
	return (r.singleCharacter && !unicode.IsPunct(c)) || unicode.IsSpace(c)
}

func (r *reader) nextWord() string {
	word := ""
	startByteIndex := r.currentByteIndex

	for byteIndex, rune := range r.text[startByteIndex:] {
		r.currentByteIndex = startByteIndex + byteIndex
		if word != "" && r.wordBoundary(r.singleCharacter, rune) {
			return word
		}

		if !unicode.IsSpace(rune) {
			word = word + string(rune)
		}
	}

	return ""
}

func (r *reader) read(s tcell.Screen, comm chan int) {
	rune, _ := utf8.DecodeRuneInString(r.text[:4])
	r.singleCharacter = r.guessSingleCharacter(rune)

	for word := r.nextWord(); word != ""; word = r.nextWord() {
		r.displayedWord = word
		spinnerInc()
		r.updateUI(s)
		r.wait(s, comm)
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

func startReader(s tcell.Screen, comm chan int) {
	fileReader := bufio.NewReader(os.Stdin)
	buf, err := io.ReadAll(fileReader)
	if err != nil {
		log.Fatalf("Could not read stdin: %v\n", err)
	}
	buf = stripByteOrderMark(buf)

	reader := reader{}
	reader.text = string(buf)
	reader.paused = true
	reader.wordsPerMinute = 300
	reader.maxByteIndex = len(reader.text)

	reader.read(s, comm)
}
