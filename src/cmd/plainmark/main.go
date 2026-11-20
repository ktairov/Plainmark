package main

import (
	"flag"
	"log"

	"plainmark/internal/desktop"
)

func main() {
	flag.Parse()
	if err := desktop.Run(flag.Arg(0)); err != nil {
		log.Fatal(err)
	}
}
