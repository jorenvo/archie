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
