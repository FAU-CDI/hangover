package headache

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// validateAddress checks if address is valid
func validateAddress(addr string) error {
	if addr == "" {
		return fmt.Errorf("empty address")
	}

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	return nil
}

const settingsWindowText = `
This program (headache) implements a GUI for hangover, the WissKI Data Viewer.  

To start the viewer, simply select the exported triplestore data and pathbuilder below.
Then click the start button below. `

// setupSettingsWindow configures the main window for the settings view
func (h *Headache) setupSettingsWindow() {
	h.handler.Stats.Log("setting up settings window")

	addr := widget.NewEntryWithData(h.settings.addr)
	addr.Validator = validateAddress
	addr.SetPlaceHolder("127.0.0.1:8000")

	images := widget.NewCheckWithData("Render Images", h.settings.images)
	html := widget.NewCheckWithData("Render HTML", h.settings.html)

	sameAs := widget.NewMultiLineEntry()
	sameAs.Bind(h.settings.sameAs)

	inverseOf := widget.NewMultiLineEntry()
	inverseOf.Bind(h.settings.inverseOf)

	quadsWidget, quadsButton := newFileSelector("Select Quads", h.w, h.settings.nquads, func(path string) error {
		if path == "" {
			return errors.New("must select a path")
		}
		return nil
	})

	pbWidget, pbButton := newFileSelector("Select Pathbuilder", h.w, h.settings.pathbuilder, func(path string) error {
		if path == "" {
			return errors.New("must select a path")
		}
		return nil
	})

	openHeadache := newDataOpener("Find Quads + Pathbuilder from folder ", h.w, h.settings.nquads, h.settings.pathbuilder)

	// setup a context to cancel the window
	_, done, cancel := h.newWindowContext(context.Background())
	var success atomic.Bool

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Data", Widget: openHeadache},
			{Widget: layout.NewSpacer()},

			{Text: "Triplestore Export", Widget: quadsButton},
			{Widget: quadsWidget, HintText: "Exported Triplestore Data to load. Typically with ending nq. "},

			{Text: "Pathbuilder Export", Widget: pbButton},
			{Widget: pbWidget, HintText: "Pathbuilder to load. Typically with ending xml. "},

			{Widget: layout.NewSpacer()},

			{Text: "SameAs", Widget: sameAs, HintText: "SameAs Predicate(s). One per line. "},
			{Text: "InverseOf", Widget: inverseOf, HintText: "InverseOf Predicate(s). One per line. "},

			{Widget: layout.NewSpacer()},

			{Widget: html, HintText: "Render HTML instead of displaying source code only"},
			{Widget: images, HintText: "Render images instead of displaying a link to the url"},

			{Widget: layout.NewSpacer()},

			{Text: "Address", Widget: addr, HintText: "Address to listen on. "},
		},
		CancelText: "Cancel",
		SubmitText: "Start Viewer",
		OnSubmit: func() {
			success.Store(true)
			cancel()
		},
	}

	// resize the window to a sensible size
	h.w.Resize(fyne.NewSize(640, 460))

	// setup the content
	h.setContent(
		container.NewVBox(
			widget.NewLabel(settingsWindowText),
			form,
		),
	)

	// force the refresh of the form
	form.Refresh()

	h.done.Add(1)
	go func() {
		defer h.done.Done()

		// wait for the cleanup to complete
		<-done
		h.handler.Stats.Log("cleanup up settings window", "success", success.Load())
		h.clearContent()

		// exit out if the user just closed the window
		if !success.Load() {
			h.w.Close()
			return
		}

		// setup the run window
		go h.setupRunWindow()
	}()
}
