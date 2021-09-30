package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"time"
	"unicode"
)

func speedRead(s string) {
	word := ""
	for _, c := range s {
		if unicode.IsSpace(c) {
			fmt.Printf("%s\n", word)
			word = ""
			time.Sleep(100 * time.Millisecond)
		} else {
			word = word + string(c)
		}
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\000')
	if err != io.EOF {
		log.Fatalf("Could not read stdin: %s\n", err)
	}

	speedRead(text)
}
