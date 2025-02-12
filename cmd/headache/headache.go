package main

import (
	"os"

	"github.com/FAU-CDI/hangover/internal/headache"
)

func main() {
	debug, ok := os.LookupEnv("HEADACHE_DEBUG")
	pain := headache.New(ok && debug != "")
	pain.RunAndWait()
}
