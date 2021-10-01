package main

import (
	"bufio"
	"github.com/gdamore/tcell/v2"
	"io"
	"log"
	"os"
	"time"
	"unicode"
	"unicode/utf8"
	"strconv"
)

func write(s tcell.Screen, word string, col int, row int) {
	for i, c := range word {
		s.SetContent(col + i, row, c, nil, tcell.StyleDefault)
	}
	s.Show()
}

func writeWord(s tcell.Screen, word string) {
	width, height := s.Size()
	write(
		s,
		word,
		width/2 - utf8.RuneCountInString(word)/2,
		height/2,
	)
}

func writeStatus(s tcell.Screen, word string) {
	width, height := s.Size()
	write(s, word, width - 8, height - 1)
}

func speedRead(s tcell.Screen, text string, comm chan int) {
	containsText := false
	word := ""
	var delayMs time.Duration = 1_000

	for _, c := range text {
		word = word + string(c)
		if containsText && (unicode.IsSpace(c) || unicode.IsPunct(c)) {
			s.Clear()
			writeStatus(s, strconv.Itoa(int(delayMs)))
			writeWord(s, word)
			word = ""
			containsText = false
			time.Sleep(delayMs * time.Millisecond)
			select {
			case msg := <-comm:
				switch msg {
				case COMM_SPEED_INC:
					delayMs -= 100
				case COMM_SPEED_DEC:
					delayMs += 100
				}
			default:
			}
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
