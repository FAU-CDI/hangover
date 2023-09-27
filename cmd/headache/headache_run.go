package main

import (
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/FAU-CDI/hangover/internal/viewer"
	"github.com/fyne-io/terminal"
)

func (h *Headache) setupRunWindow() {
	// setup the window
	h.done.Add(1)

	t := terminal.New()
	go func() {
		defer h.done.Done()
		t.RunWithConnection(os.Stdin, os.Stderr)
	}()

	h.W.SetContent(t)

	/*
		layout := container.NewVBox(info, button, grid.tg)
		ui.W.SetContent(layout)

		context, cancel := context.WithCancel(context.Background())
		done := runHangover(context, io.MultiWriter(grid, os.Stderr), ui)
		ui.W.SetOnClosed(func() {
			cancel()
			<-done
			a.Quit()
		})
		ui.W.Show()
	*/

	// start and run the viewer
	h.runViewer(t)
}

// runViewer
func (h *Headache) runViewer(out io.Writer) (addr string, err error) {
	h.done.Add(2)

	listener, err := net.Listen("tcp", h.settings.Addr())
	if err != nil {
		return "", err
	}

	// create a new writer
	handler := viewer.NewViewer(out)
	go func() {
		defer h.done.Done()
		http.Serve(listener, handler)
	}()

	// start the actual process of reading stuff
	go func() {
		defer handler.Close()
		defer h.done.Done()

		time.Sleep(time.Second)
		io.WriteString(out, "hello world\n")
		time.Sleep(time.Second)
		io.WriteString(out, "hello world\n")
	}()

	return listener.Addr().String(), nil
}
