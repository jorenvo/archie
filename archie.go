package main

import (
	"bufio"
	"fmt"
	"github.com/jroimartin/gocui"
	"io"
	"log"
	"os"
	"time"
	"unicode"
)

func getMainView(g *gocui.Gui) *gocui.View {
	view, err := g.View("hello")
	if err != nil {
		log.Fatalf("Error getting view: %s\n", err)
	}
	return view
}

func writeWord(word string) func(*gocui.Gui) error {
	return func(g *gocui.Gui) error {
		view := getMainView(g)
		view.Clear()
		fmt.Fprintf(view, "%s", word)
		return nil
	}
}

func speedRead(g *gocui.Gui, s string) {
	containsText := false
	word := ""
	for _, c := range s {
		word = word + string(c)
		if containsText && (unicode.IsSpace(c) || unicode.IsPunct(c)) {
			fmt.Println(word)
			g.Update(writeWord(word))
			word = ""
			containsText = false
			return;
			time.Sleep(1000 * time.Millisecond)
		} else {
			containsText = true
		}
	}
}

func mainReader(g *gocui.Gui) {
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\000')
	if err != io.EOF {
		log.Fatalf("Could not read stdin: %s\n", err)
	}

	speedRead(g, text)
}

func layout(g *gocui.Gui) error {
	const size = 40
	maxX, maxY := g.Size()
	if v, err := g.SetView("hello", maxX/2-size/2, maxY/2, maxX/2+size/2, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(v, "Hello world!")
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func main() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	go mainReader(g)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}
