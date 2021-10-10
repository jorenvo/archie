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
	displayedWordIndex      int
	debug                   string
	searching               bool
	searchStartRuneIndex    int
	currentSearch           []rune
	singleCharacter         bool
	context                 bool
	currentRuneIndex        int
	maxRuneIndex            int
	newWordsPerMinuteBuffer int
}

func (r *reader) writeMiddleWithContext() {
	const fillRatio = 0.8
	const maxGuessSize = 100
	guessEnd := min(len(r.text), r.currentRuneIndex+maxGuessSize)
	nextCharacters := string(r.text[r.currentRuneIndex:guessEnd])
	width := r.screen.width() / r.screen.runeWidthString(nextCharacters)
	charsContext := int(float64(width) * fillRatio)

	wordCenter := r.displayedWordIndex + len([]rune(r.displayedWord))/2
	left := wordCenter
	right := wordCenter

	// Expand context left and right. If not possible try to expand
	// on the other side.
	for i := 0; i < charsContext/2; i++ {
		if left > 0 {
			left--
		} else if right < len(r.text) {
			right++
		}

		if right < len(r.text) {
			right++
		} else if left > 0 {
			left--
		}
	}

	var wordAndContext []rune
	for _, rune := range r.text[left : right+1] {
		if unicode.IsPrint(rune) {
			wordAndContext = append(wordAndContext, rune)
		} else {
			// Skip \r (to avoid double space with DOS line endings).
			if rune == '\r' {
				continue
			}

			// Replace non-printable characters with a space. Usually
			// we're replacing a \n.
			wordAndContext = append(wordAndContext, ' ')
		}
	}
	r.screen.writeWord(string(wordAndContext))
}

func (r *reader) writeMiddle(unit string) {
	if r.newWordsPerMinuteBuffer == 0 {
		if r.context {
			r.writeMiddleWithContext()
		} else {
			r.screen.writeWord(r.displayedWord)
		}
	} else {
		r.screen.writeWord(fmt.Sprintf("New %s per min: %d", unit, r.newWordsPerMinuteBuffer))
	}
}

func (r *reader) updateUI() {
	r.screen.clear()

	r.screen.write(r.debug, 0, 0)

	unit := "words"
	if r.singleCharacter {
		unit = "characters"
	}

	// This should be part of reader
	r.screen.writeStatus(
		fmt.Sprintf("%d %s per min", r.wordsPerMinute, unit),
		r.searching,
		string(r.currentSearch),
		r.paused,
		r.currentRuneIndex,
		r.maxRuneIndex,
	)

	r.writeMiddle(unit)
}

func (r *reader) search(nextOccurrence bool) {
	searchStart := r.searchStartRuneIndex
	if nextOccurrence {
		searchStart = r.currentRuneIndex
	}

	for ; searchStart < len(r.text); searchStart++ {
		for searchIndex, searchRune := range r.currentSearch {
			textRune := r.text[searchStart+searchIndex]
			if textRune != searchRune {
				break
			}

			if searchIndex == len(r.currentSearch)-1 {
				r.currentRuneIndex = searchStart
				r.displayedWord, r.displayedWordIndex = r.nextWord()
				r.debug = fmt.Sprintf("%v", r.currentRuneIndex)
				return
			}
		}
	}
}

func (r *reader) handleCommsSearch(comm chan int, commSearch chan rune) bool {
	messagesPending := true
	handledMessage := false
	for messagesPending {
		select {
		case char := <-commSearch:
			handledMessage = true
			r.context = true
			r.currentSearch = append(r.currentSearch, char)
			r.search(false)
		default:
			messagesPending = false
		}
	}

	messagesPending = true
	for messagesPending {
		select {
		case msg := <-comm:
			handledMessage = true
			switch msg {
			case COMM_SEARCH:
				r.search(true)
			case COMM_CONFIRM:
				r.searching = false
				r.context = false
				r.currentSearch = nil
			case COMM_BACKSPACE:
				newLen := max(0, len(r.currentSearch)-1)
				r.currentSearch = r.currentSearch[:newLen]
				r.search(false)
			}
		default:
			messagesPending = false
		}
	}

	return handledMessage
}

func (r *reader) handleCommsWpm(comm chan int, commSearch chan rune) bool {
	messagesPending := true
	for messagesPending {
		// flush commSearch
		select {
		case <-commSearch:
		default:
			messagesPending = false
		}
	}

	handledMessage := false
	messagesPending = true
	for messagesPending {
		select {
		case msg := <-comm:
			handledMessage = true
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
		default:
			messagesPending = false
		}
	}

	return handledMessage
}

func (r *reader) handleCommsRegular(comm chan int, commSearch chan rune) bool {
	messagesPending := true
	for messagesPending {
		// flush commSearch
		select {
		case <-commSearch:
		default:
			messagesPending = false
		}
	}

	const speedInc = 5
	handledMessage := false
	messagesPending = true
	for messagesPending {
		select {
		case msg := <-comm:
			handledMessage = true
			switch {
			case msg >= COMM_DIGIT_0 && msg <= COMM_DIGIT_9:
				r.newWordsPerMinuteBuffer =
					r.newWordsPerMinuteBuffer*10 + msg - COMM_DIGIT_0
				r.paused = true
			case msg == COMM_SEARCH:
				r.searching = true
				r.searchStartRuneIndex = r.currentRuneIndex
			case msg == COMM_SPEED_INC:
				r.wordsPerMinute += speedInc
			case msg == COMM_SPEED_DEC:
				r.wordsPerMinute -= speedInc
			case msg == COMM_TOGGLE:
				r.paused = !r.paused
			case msg == COMM_SINGLE_CHARACTER:
				r.singleCharacter = !r.singleCharacter
			case msg == COMM_SENTENCE_BACKWARD:
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
						r.displayedWord, r.displayedWordIndex = r.nextWord()
						break
					} else {
						r.currentRuneIndex = previousBreak
						r.currentRuneIndex++
						r.displayedWord, r.displayedWordIndex = r.nextWord()
					}
				}
			case msg == COMM_SENTENCE_FORWARD:
				// Skip this character in case it's a period
				r.currentRuneIndex++
				nextBreak := indexAnyRune(r.text[r.currentRuneIndex:], sentenceBreaks)
				if nextBreak == -1 {
					break
				}

				// += because IndexAny ran on substring
				r.currentRuneIndex += nextBreak

				r.currentRuneIndex++
				r.displayedWord, r.displayedWordIndex = r.nextWord()
			}
		default:
			messagesPending = false
		}
	}

	return handledMessage
}

func (r *reader) handleComms(comm chan int, commSearch chan rune) bool {
	if r.searching {
		return r.handleCommsSearch(comm, commSearch)
	}

	if r.newWordsPerMinuteBuffer != 0 {
		return r.handleCommsWpm(comm, commSearch)
	}

	return r.handleCommsRegular(comm, commSearch)
}

// Waits but still handles comms at 60 Hz
func (r *reader) wait(comm chan int, commSearch chan rune, timeMs int64) {
	const Hz = 60
	remainingMs := timeMs

	for remainingMs > 0 {
		prevTime := time.Now().UnixMilli()

		if r.handleComms(comm, commSearch) {
			r.updateUI()
			remainingMs = timeMs
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

func (r *reader) nextWord() (word string, startIndex int) {
	startIndex = -1
	word = ""

	for ; r.currentRuneIndex < len(r.text); r.currentRuneIndex++ {
		rune := r.text[r.currentRuneIndex]
		if word != "" && r.wordBoundary(r.singleCharacter, rune) {
			return word, startIndex
		}

		if !unicode.IsSpace(rune) {
			word = word + string(rune)
			if startIndex == -1 {
				startIndex = r.currentRuneIndex
			}
		}
	}

	return "", -1
}

func (r *reader) getDelayMs() int64 {
	return int64(math.Round(1_000 / (float64(r.wordsPerMinute) / float64(60))))
}

func (r *reader) getBlankRatio() float64 {
	if r.singleCharacter && !r.context {
		return 0.2 // TODO: is this too fast or slow?
	} else {
		return 0
	}
}

func (r *reader) read(comm chan int, commSearch chan rune) {
	r.singleCharacter = r.guessSingleCharacter(r.text[0])

	for {
		r.displayedWord, r.displayedWordIndex = r.nextWord()
		if r.displayedWord == "" {
			break
		}

		blankRatio := r.getBlankRatio()
		delayMs := float64(r.getDelayMs())
		wordMs := int64(delayMs * (1.0 - blankRatio))
		blankMs := int64(delayMs * blankRatio)

		spinnerInc()
		r.updateUI()
		r.wait(comm, commSearch, wordMs)

		if blankMs > 0 {
			r.screen.clearWord()
			r.wait(comm, commSearch, blankMs)
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

func startReader(s screen, comm chan int, commSearch chan rune) {
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

	reader.read(comm, commSearch)
}
