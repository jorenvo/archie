package main

import (
	"bufio"
	// "fmt"
	"github.com/gdamore/tcell/v2"
	"io"
	"log"
	"os"
	"time"
	"unicode"
)

func writeWord(s tcell.Screen, word string) {
	s.Clear()
	for i, c := range word {
		s.SetContent(i, 0, c, nil, tcell.StyleDefault)		
	}
	s.Show()
}

func speedRead(s tcell.Screen, text string) {
	containsText := false
	word := ""
	for _, c := range text {
		word = word + string(c)
		if containsText && (unicode.IsSpace(c) || unicode.IsPunct(c)) {
			writeWord(s, word)
			word = ""
			containsText = false
			time.Sleep(1000 * time.Millisecond)
		} else {
			containsText = true
		}
	}
}

func mainReader(s tcell.Screen) {
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\000')
	if err != io.EOF {
		log.Fatalf("Could not read stdin: %s\n", err)
	}

	speedRead(s, text)
}

func drawUpdateCounter(events int, s tcell.Screen) {
	for i := 0; i < events; i++ {
		s.SetContent(i, 0, tcell.RuneHLine, nil, tcell.StyleDefault)
	}
}

func quit(s tcell.Screen) {
	s.Fini()
	os.Exit(0)
}

func main() {
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalln(err)
	}
	if err := s.Init(); err != nil {
		log.Fatalln(err)
	}

	go mainReader(s)

	for {
		s.Show()

		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape, tcell.KeyEnter, tcell.KeyCtrlC:
				quit(s)
			}
		}
	}
}
