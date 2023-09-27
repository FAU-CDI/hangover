package main

import (
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

// Headache implements the headache UI
type Headache struct {
	used atomic.Bool

	A fyne.App    // the app used by headache
	W fyne.Window // the window used by headache

	done sync.WaitGroup // WaitGroup for closing

	// settings holds the settings of the server
	settings Settings
}

// NewHeadache setups a new Headache application
func NewHeadache() *Headache {
	// create a new app and window
	a := app.New()
	w := a.NewWindow("Headache")

	// and return the app along with the settings
	return &Headache{
		A: a,
		W: w,

		settings: NewSettings(),
	}
}

// Close closes all windows
func (h *Headache) Close() {
	h.W.Close()
}

// RunAndWait runs the app and waits for it to complete
func (h *Headache) RunAndWait() {
	if !h.used.CompareAndSwap(false, true) {
		panic("RunAndWait already called")
	}

	// setup the main window
	h.setupSettingsWindow()

	// show and run the window, then wait
	h.W.ShowAndRun()
	h.done.Wait()
}
