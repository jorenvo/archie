package main

import (
	// "fmt"
	"github.com/gdamore/tcell/v2"
	"log"
	"os"
)

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
		// EventResize events happen every time a Show happens. Avoid
		// calling an expensive Sync here.
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape, tcell.KeyEnter, tcell.KeyCtrlC:
				quit(s)
			}
		}
	}
}
