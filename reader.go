package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"time"
	"unicode"
)

type reader struct {
	screen                  screen
	text                    []rune
	paused                  bool
	wordsPerMinute          int
	displayedWord           string
	debug                   string
	singleCharacter         bool
	currentRuneIndex        int
	maxRuneIndex            int
	newWordsPerMinuteBuffer int
}

func (r *reader) updateUI() {
	r.screen.clear()

	r.screen.write(r.debug, 0, 0)

	unit := "words"
	if r.singleCharacter {
		unit = "characters"
	}
	r.screen.writeStatus(
		fmt.Sprintf("%d %s per min", r.wordsPerMinute, unit),
		r.paused,
		r.currentRuneIndex,
		r.maxRuneIndex,
	)

	if r.newWordsPerMinuteBuffer == 0 {
		r.screen.writeWord(r.displayedWord)
	} else {
		r.screen.writeWord(fmt.Sprintf("New %s per min: %d", unit, r.newWordsPerMinuteBuffer))
	}
}

type skipPastCharacterParam bool

const (
	Backwards skipPastCharacterParam = true
	Forwards                         = false
)

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
				startingByteIndex := r.currentRuneIndex
				for startingByteIndex == r.currentRuneIndex {
					for i := 0; i < skippedCharactersBackwards; i++ {
						if r.currentRuneIndex > 0 {
							r.currentRuneIndex--
						}
					}
					skippedCharactersBackwards++

					previousBreak := lastIndexAnyRune(r.text[:r.currentRuneIndex], sentenceBreaks)
					if previousBreak == -1 {
						// No break found, go back to the beginning of file
						r.currentRuneIndex = 0
						r.displayedWord = r.nextWord()
						break
					} else {
						r.currentRuneIndex = previousBreak
						r.currentRuneIndex++
						r.displayedWord = r.nextWord()
					}
				}
			case COMM_SENTENCE_FORWARD:
				// Skip this character in case it's a period
				r.currentRuneIndex++
				nextBreak := indexAnyRune(r.text[r.currentRuneIndex:], sentenceBreaks)
				if nextBreak == -1 {
					break
				}

				// += because IndexAny ran on substring
				r.currentRuneIndex += nextBreak

				r.currentRuneIndex++
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
func (r *reader) wait(comm chan int) {
	const Hz = 60
	remainingMs := r.getDelayMs()

	for remainingMs > 0 {
		prevTime := time.Now().UnixMilli()

		if r.handleComms(comm) {
			r.updateUI()
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
	return r.screen.runeWidth(c) == 2
}

func (r *reader) wordBoundary(singleCharacter bool, c rune) bool {
	// TODO: IsPunct will include quotes
	return (r.singleCharacter && !unicode.IsPunct(c)) || unicode.IsSpace(c)
}

func (r *reader) nextWord() string {
	word := ""

	for ; r.currentRuneIndex < len(r.text); r.currentRuneIndex++ {
		rune := r.text[r.currentRuneIndex]
		if word != "" && r.wordBoundary(r.singleCharacter, rune) {
			return word
		}

		if !unicode.IsSpace(rune) {
			word = word + string(rune)
		}
	}

	return ""
}

func (r *reader) read(comm chan int) {
	r.singleCharacter = r.guessSingleCharacter(r.text[0])

	for word := r.nextWord(); word != ""; word = r.nextWord() {
		r.displayedWord = word
		spinnerInc()
		r.updateUI()
		r.wait(comm)
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

func startReader(s screen, comm chan int) {
	fileReader := bufio.NewReader(os.Stdin)
	buf, err := io.ReadAll(fileReader)
	if err != nil {
		s.error(err)
	}

	if len(buf) == 0 {
		s.error(errors.New("No content"))
	}

	buf = stripByteOrderMark(buf)

	reader := reader{}
	reader.screen = s
	reader.text = []rune(string(buf))
	reader.paused = true
	reader.wordsPerMinute = 300
	reader.maxRuneIndex = len(reader.text)

	reader.read(comm)
}
