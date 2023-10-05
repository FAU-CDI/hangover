package headache

import (
	"context"
	"io"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"github.com/FAU-CDI/hangover"
	"github.com/FAU-CDI/hangover/internal/console"
	"github.com/FAU-CDI/hangover/internal/viewer"
)

// Headache implements the headache UI
type Headache struct {
	used atomic.Bool

	a fyne.App    // the app used by headache
	w fyne.Window // the window used by headache

	m *container.TabItem // the main tab
	g *container.TabItem // the grid tab

	// the handler and grid to be used later
	handler *viewer.Viewer
	console *console.Console

	done sync.WaitGroup // WaitGroup for closing

	settings settings // settings holds the settings of the server
	status   status   // status holds the current status
}

var icon = &fyne.StaticResource{
	StaticName:    "Icon.svg",
	StaticContent: hangover.IconSVG,
}

// New setups a new Headache application
func New() *Headache {

	// create a new app and window
	var h Headache

	h.a = app.NewWithID("de.fau.data.wisski.headache")
	h.a.SetIcon(icon)

	h.w = h.a.NewWindow("Headache")
	h.w.SetIcon(icon)

	// note(twiesing): It is critical to create these after the app
	h.settings = newSettings()
	h.status = newStatus()

	h.m = container.NewTabItem("Overview", layout.NewSpacer())

	h.console = console.New(0)
	h.g = container.NewTabItem("Console", h.console.CanvasObject())

	h.w.SetContent(
		container.NewAppTabs(h.m, h.g),
	)

	h.handler = viewer.NewViewer(io.MultiWriter(h.console, os.Stderr))
	h.handler.Stats.OnUpdate = h.status.Set

	return &h
}

// RunAndWait runs the app and waits for it to complete
func (h *Headache) RunAndWait() {
	if !h.used.CompareAndSwap(false, true) {
		panic("RunAndWait already called")
	}

	// setup the main window
	h.setupSettingsWindow()

	// show and run the window, then wait
	h.w.ShowAndRun()
	h.handler.Stats.Log("application exited, waiting on cleanup")
	h.done.Wait()
}

func (h *Headache) setContent(o fyne.CanvasObject) {
	h.m.Content = o
}
func (h *Headache) clearContent() {
	h.setContent(layout.NewSpacer())
}

// newWindowContext creates a new context that is cancelled when one of the following three things occurs:
// - the SIGINT signal is received
// - the window is closed
// - the cancel function is called
// The context is automatically cleaned up once it is cancelled, and no calling the cancel function is needed.
func (h *Headache) newWindowContext(parent context.Context) (ctx context.Context, done chan struct{}, cancel context.CancelFunc) {
	done = make(chan struct{})

	ctx1, cancel1 := context.WithCancel(parent)
	ctx2, cancel2 := signal.NotifyContext(ctx1, syscall.SIGINT)

	ctx = ctx2
	cancel = func() {
		defer cancel1()
		defer cancel2()
	}

	h.handler.Stats.LogDebug("setting up intercepts")
	h.w.SetCloseIntercept(func() {
		cancel()
		h.w.Close()
	})

	h.done.Add(1)
	go func() {
		defer h.done.Done()
		defer close(done)

		// wait for context to close and cleanup
		<-ctx.Done()
		cancel()

		// reset the handler
		h.handler.Stats.LogDebug("clearing intercepts")
		h.w.SetCloseIntercept(h.w.Close)
	}()

	return
}
