package main

import (
	"os"

	"github.com/FAU-CDI/hangover/internal/wisski2/wca"
)

func main() {
	if len(os.Args) < 2 {
		panic("needs path")
	}

	// create a new archive
	archive, err := wca.CreateArchive(os.Args[1], nil)
	if err != nil {
		panic(err)
	}

	// got the archive
	defer archive.Close()
}
