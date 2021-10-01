package main

import (
	"bufio"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"io"
	"log"
	"math"
	"os"
	"time"
	"unicode"
	"unicode/utf8"
)

// TODO global variables?
var wordsPerMinute int = 300
var displayedWord string = ""

func write(s tcell.Screen, word string, col int, row int) {
	for i, c := range word {
		s.SetContent(col+i, row, c, nil, tcell.StyleDefault)
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

func writeStatus(s tcell.Screen, word string) {
	width, height := s.Size()
	write(s, word, width-utf8.RuneCountInString(word), height-1)
}

func updateUI(s tcell.Screen) {
	s.Clear()
	writeStatus(s, fmt.Sprintf("%d words per min", wordsPerMinute))
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
			}
			handledMessage = true
		default:
			messagesPending = false
		}
	}

	return handledMessage
}

// TODO: this is available in go v1.17
func unixMilli() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func getDelayMs() int64 {
	return int64(math.Round(1_000 / (float64(wordsPerMinute) / float64(60))))
}

// Waits but still handles comms at 60 Hz
func sleep(s tcell.Screen, comm chan int) {
	const Hz = 60
	remainingMs := getDelayMs()

	for remainingMs > 0 {
		prevTime := unixMilli()

		if handleComms(comm) {
			updateUI(s)
		}

		time.Sleep(1_000 / Hz * time.Millisecond)
		remainingMs -= unixMilli() - prevTime
	}
}

func speedRead(s tcell.Screen, text string, comm chan int) {
	containsText := false
	word := ""

	for _, c := range text {
		word = word + string(c)
		if containsText && (unicode.IsSpace(c) || unicode.IsPunct(c)) {
			displayedWord = word
			word = ""

			updateUI(s)

			containsText = false
			sleep(s, comm)
		} else {
			containsText = true
		}
	}
}

func mainReader(s tcell.Screen, comm chan int) {
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\000')
	if err != io.EOF {
		log.Fatalf("Could not read stdin: %s\n", err)
	}

	speedRead(s, text, comm)
}
