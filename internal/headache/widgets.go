package headache

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/FAU-CDI/hangover"
)

// newDataOpener creates a new button that selects the nquads and pathbuilder from the given folder.
func newDataOpener(label string, parent fyne.Window, vNq, vPB binding.String) *widget.Button {
	b := widget.NewButton(label, func() {
		dialog.ShowFolderOpen(func(uc fyne.ListableURI, err error) {
			if err != nil || uc == nil || uc.Scheme() != "file" {
				return
			}

			nq, pb, _, err := hangover.FindSource(false, uc.Path())
			if err != nil {
				return
			}
			vNq.Set(nq)
			vPB.Set(pb)

		}, parent)
	})
	return b
}

// newFileSelector creates a two new widgets, a readonly entry and a file selector
func newFileSelector(label string, parent fyne.Window, v binding.String, validator func(path string) error) (*widget.Entry, *widget.Button) {
	w := widget.NewEntryWithData(v)
	w.Validator = validator

	b := widget.NewButton(label, func() {
		dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
			if err != nil || uc == nil {
				return
			}

			uri := uc.URI()
			if uri == nil || uri.Scheme() != "file" {
				return
			}
			w.SetText(uri.Path())
			w.Validate()
		}, parent)
	})

	return w, b
}

type WrappedValidator struct {
	*fyne.Container
	validate func() error
}

func (ww WrappedValidator) Validate() error {
	return ww.validate()
}
