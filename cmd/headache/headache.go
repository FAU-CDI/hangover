package main

import "github.com/FAU-CDI/hangover/internal/headache"

func main() {
	pain := headache.New()
	pain.RunAndWait()
}
