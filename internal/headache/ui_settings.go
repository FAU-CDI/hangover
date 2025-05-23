//spellchecker:words headache
package headache

//spellchecker:words context errors http strconv strings sync atomic fyne container layout widget github pkglib
import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/tkw1536/pkglib/fsx"
)

const settingsWindowText = `
This program (headache) implements a GUI for hangover, the WissKI Data Viewer.  

To start the viewer, simply select the exported triplestore data and pathbuilder below.
Then click the start button below (you may have to scroll to the bottom). `

var (
	errUnableToDownloadPathbuilder = errors.New("unable to download pathbuilder")
)

// setupSettingsWindow configures the main window for the settings view.
func (h *Headache) setupSettingsWindow() {
	h.handler.Stats.Log("setting up settings window")

	addr := widget.NewEntryWithData(h.settings.addr)
	addr.Validator = validateAddress
	addr.SetPlaceHolder("127.0.0.1:8000")

	images := widget.NewCheckWithData("Render Images", h.settings.images)
	html := widget.NewCheckWithData("Render HTML", h.settings.html)

	tipsy := widget.NewEntryWithData(h.settings.tipsy)
	tipsy.Validator = isValidTipsy

	public := widget.NewMultiLineEntry()
	public.Bind(h.settings.public)

	sameAs := widget.NewMultiLineEntry()
	sameAs.Bind(h.settings.sameAs)

	inverseOf := widget.NewMultiLineEntry()
	inverseOf.Bind(h.settings.inverseOf)

	quadsWidget, quadsButton := newFileSelector("Select '.nq' File", h.w, h.settings.nquads, isFile)
	pbWidget, pbButton := newFileSelector("Select '.xml' File", h.w, h.settings.pathbuilder, func(path string) (e error) {
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
			if err != nil {
				return fmt.Errorf("unable to create pathbuilder request: %w", err)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("%w %q: %w", errUnableToDownloadPathbuilder, path, err)
			}
			defer func() {
				if e2 := res.Body.Close(); e2 != nil {
					e2 = fmt.Errorf("failed to close body: %w", e2)
					if e == nil {
						e = e2
					} else {
						e = errors.Join(e, e2)
					}
				}
			}()

			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("%w %q: expected code %d, but got %d", errUnableToDownloadPathbuilder, path, http.StatusOK, res.StatusCode)
			}

			return nil
		}
		return isFile(path)
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
			{Widget: quadsWidget, HintText: "Exported Triplestore Data to load, a path to an '.nq' file. "},

			{Text: "Pathbuilder Export", Widget: pbButton},
			{Widget: pbWidget, HintText: "Pathbuilder to load. A url to download it from, or path to an '.xml' file. "},

			{Widget: layout.NewSpacer()},

			{Text: "SameAs", Widget: sameAs, HintText: "SameAs Predicate(s). One per line. "},
			{Text: "InverseOf", Widget: inverseOf, HintText: "InverseOf Predicate(s). One per line. "},

			{Widget: layout.NewSpacer()},

			{Text: "Public URLs", Widget: public, HintText: "Public URL(s) to replace with viewer content. One per line. "},
			{Widget: html, HintText: "Render HTML instead of displaying source code only"},
			{Widget: images, HintText: "Render images instead of displaying a link to the url"},

			{Widget: layout.NewSpacer()},

			{Widget: tipsy, HintText: "Embed a TIPSY instance from the given URL"},

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
		container.NewScroll(
			container.NewVBox(
				widget.NewLabel(settingsWindowText),
				form,
			),
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

var (
	errEmptyAddress = errors.New("empty address")
)

// validateAddress checks if address is valid.
func validateAddress(addr string) error {
	if addr == "" {
		return errEmptyAddress
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

var (
	errNoFilePath = errors.New("no file path provided")
	errNotAFile   = errors.New("not a file")
)

// isFile validates that path is a file.
//
// If path is not a file, returns an error.
// Otherwise, returns nil.
func isFile(path string) error {
	if path == "" {
		return errNoFilePath
	}

	ok, err := fsx.IsRegular(path, true)
	if err != nil {
		return fmt.Errorf("%w: %q: %w", errNotAFile, path, err)
	}
	if !ok {
		return fmt.Errorf("%w: %q", errNotAFile, path)
	}
	return nil
}

var errInvalidURL = errors.New("URL must be empty or start with 'http://' or 'https://'")

func isValidTipsy(url string) error {
	if url == "" || strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return nil
	}

	return errInvalidURL
}
