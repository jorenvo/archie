package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
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
	currentByteIndex        int // TODO: rename now we're using runes
	maxByteIndex            int // TODO: rename now we're using runes
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
		r.currentByteIndex,
		r.maxByteIndex,
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
				startingByteIndex := r.currentByteIndex
				for startingByteIndex == r.currentByteIndex {
					for i := 0; i < skippedCharactersBackwards; i++ {
						if r.currentByteIndex > 0 {
							r.currentByteIndex--
						}
					}
					skippedCharactersBackwards++

					previousBreak := lastIndexAnyRune(r.text[:r.currentByteIndex], sentenceBreaks)
					r.debug = fmt.Sprintf("byte index at: %d, previous break at: %d", r.currentByteIndex, previousBreak)
					if previousBreak == -1 {
						// No break found, go back to the beginning of file
						r.currentByteIndex = 0
						r.displayedWord = r.nextWord()
						break
					} else {
						r.currentByteIndex = previousBreak
						r.currentByteIndex++
						r.displayedWord = r.nextWord()
					}
				}
			case COMM_SENTENCE_FORWARD:
				// Skip this character in case it's a period
				r.currentByteIndex++
				nextBreak := indexAnyRune(r.text[r.currentByteIndex:], sentenceBreaks)
				if nextBreak == -1 {
					break
				}

				// += because IndexAny ran on substring
				r.currentByteIndex += nextBreak

				r.currentByteIndex++
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
	if err != nil || len(buf) == 0 {
		log.Fatalf("Could not read stdin: %v\n", err)
	}
	buf = stripByteOrderMark(buf)

	reader := reader{}
	reader.screen = s
	reader.text = []rune(string(buf))
	reader.paused = true
	reader.wordsPerMinute = 300
	reader.maxByteIndex = len(reader.text)

	reader.read(comm)
}
