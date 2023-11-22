package headache

import (
	"context"
	"net"
	"net/http"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/FAU-CDI/hangover/internal/glass"
	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/pkg/browser"
)

const runWindowText = `
Hangover is now loading your dataset, and then starting a local server to display it.
You can watch the progress below.

Click the "Open In Browser" button to view the interface.
Close this window to stop the server.
`

func (h *Headache) setupRunWindow() {
	// start the listener
	listener, err := net.Listen("tcp", h.settings.Addr())
	if err != nil {
		h.handler.Stats.LogError("http.Serve", err, "addr", h.settings.Addr())
		return
	}

	// create a button to open the viewer
	url := "http://" + listener.Addr().String()
	button := widget.NewButton("Open In Default Browser", func() {
		browser.OpenURL(url)
	})

	layout := container.NewVBox(
		widget.NewLabel(runWindowText),
		container.NewHBox(
			widget.NewLabel("Server running at: "), widget.NewLabel(url), button,
		),
		container.NewHBox(
			widget.NewLabel("Current Stage: "),
			widget.NewLabelWithData(h.status.stage),
		),
		h.status.bar,
		container.New(
			layout.NewGridLayout(2),

			widget.NewLabel("Direct Triples"), widget.NewLabelWithData(h.status.directTriples),
			widget.NewLabel("Datum Triples"), widget.NewLabelWithData(h.status.datumTriples),
			widget.NewLabel("Masked Predicate Triples"), widget.NewLabelWithData(h.status.maskedPredTriples),
			widget.NewLabel("Masked Data Triples"), widget.NewLabelWithData(h.status.maskedDataTriples),
			widget.NewLabel("Inverse Triples"), widget.NewLabelWithData(h.status.inverseTriples),
			widget.NewLabel("Conflict Triples"), widget.NewLabelWithData(h.status.conflictTriples),
		),
	)

	// setup the ox
	h.setContent(container.NewScroll(layout))

	// create a context and run it!
	context, _, cancel := h.newWindowContext(context.Background())
	h.runViewer(listener, context, cancel)
}

// runViewer
func (h *Headache) runViewer(listener net.Listener, ctx context.Context, cancel context.CancelFunc) {
	// create a new handler
	h.done.Add(1)
	go func() {
		defer h.done.Done()

		h.handler.Stats.Log("http.Serve", "address", listener.Addr().String())
		err := http.Serve(listener, h.handler)
		h.handler.Stats.LogError("http.Serve", err)
	}()

	// start the actual process of reading stuff
	h.done.Add(1)
	go func() {
		defer h.done.Done() // we're done cleaning up

		defer listener.Close()  // close the listener
		defer h.handler.Close() // kill the cache

		h.handler.RenderFlags = h.settings.Flags()
		pb, nq := h.settings.Pathbuilder(), h.settings.Nquads()

		// create the glass by indexing
		h.handler.Stats.Log("loading files", "pathbuilder", pb, "nquads", nq)
		drincw, err := glass.Create(pb, nq, "", h.handler.RenderFlags, h.handler.Stats)
		if err != nil {
			h.handler.Stats.LogError("unable to load dataset", err)
			return
		}

		// prepare the handler
		h.handler.Stats.DoStage(stats.StageHandler, func() error {
			h.handler.Prepare(drincw.Cache, &drincw.Pathbuilder)
			return nil
		})

		// wait for the context to close
		h.handler.Stats.Log("dataset loaded")
		<-ctx.Done()
		h.handler.Stats.Log("cancelation received")
		listener.Close()
	}()
}
