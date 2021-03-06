package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"log"
	"os"
)

func quitErr(s tcell.Screen, err string) {
	s.Fini()
	fmt.Printf("ERROR: %v\n", err)
	os.Exit(1)
}

func quit(s tcell.Screen) {
	s.Fini()
	os.Exit(0)
}

func debugFile() *os.File {
	f, err := os.Create("/tmp/archie_debug.log")
	if err != nil {
		log.Fatalln(err)
	}

	return f
}

func main() {
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalln(err)
	}
	if err := s.Init(); err != nil {
		log.Fatalln(err)
	}

	screen := screen{}
	screen.tcellScreen = s
	comm := make(chan int, 64)
	commSearch := make(chan rune, 64)
	go startReader(screen, comm, commSearch)

	debugFile := debugFile()
	defer debugFile.Close()

	for {
		s.Show()

		ev := s.PollEvent()

		// EventResize events happen every time a Show happens. Avoid
		// calling an expensive Sync here.
		switch ev := ev.(type) {
		case *tcell.EventResize:
			comm <- COMM_RESIZE
		case *tcell.EventError:
			quitErr(s, ev.Error())
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape, tcell.KeyCtrlC:
				quit(s)
			case tcell.KeyEnter:
				comm <- COMM_CONFIRM
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				comm <- COMM_BACKSPACE
			case tcell.KeyLeft:
				comm <- COMM_SENTENCE_BACKWARD
			case tcell.KeyRight:
				comm <- COMM_SENTENCE_FORWARD
			case tcell.KeyCtrlS:
				comm <- COMM_SEARCH
			case tcell.KeyRune:
				rune := ev.Rune()
				commSearch <- rune
				switch rune {
				case ' ':
					comm <- COMM_TOGGLE
				case '+':
					comm <- COMM_SPEED_INC
				case '-':
					comm <- COMM_SPEED_DEC
				case 'w':
					comm <- COMM_SINGLE_CHARACTER
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					comm <- COMM_DIGIT_0 + int(rune) - int('0')
				}
			}
		}
	}
}
