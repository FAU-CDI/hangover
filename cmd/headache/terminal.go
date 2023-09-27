package main

/*

func (opt *Headache) addr() string {
	addr, _ := opt.Addr.Get()
	return "http://" + addr
}

func runHeadacheUI(a fyne.App, ui *Headache) {

	addr := ui.addr()
	button := widget.NewButton("Open Viewer In Browser", func() {
		browser.OpenURL(addr)
	})

	info := widget.NewLabel("The WissKI Viewer has been started at " + addr + ".\nClick the button to open it in a browser.\nClose the window to quit. ")

	grid := NewWriterGrid(80, 25)
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
}

func runHangover(ctx context.Context, writer io.Writer, ui *Headache) chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)

		// setup handler
		var handler = viewer.NewViewer(writer)

		var flags viewer.RenderFlags

		sa, _ := ui.SameAs.Get()
		io, _ := ui.InverseOf.Get()
		sparkl.ParsePredicateString(&flags.Predicates.SameAs, sa)
		sparkl.ParsePredicateString(&flags.Predicates.InverseOf, io)

		flags.ImageRender, _ = ui.Images.Get()
		flags.HTMLRender, _ = ui.HTML.Get()

		handler.RenderFlags = flags

		bind, _ := ui.Addr.Get()

		listener, err := net.Listen("tcp", bind)
		if err != nil {
			handler.Stats.LogError("listen", err)
			return
		}
		defer listener.Close()

		// start serving the handler
		go func() {
			http.Serve(listener, handler)
		}()

		pb, _ := ui.PBPath.Get()
		nq, _ := ui.QuadsPath.Get()

		drincw, err := glass.Create(pb, nq, "", flags, handler.Stats)
		if err != nil {
			handler.Stats.LogError("listen", err)
			return
		}

		// otherwise create a viewer
		defer handler.Close()

		handler.Stats.DoStage(stats.StageHandler, func() error {
			handler.Prepare(drincw.Cache, &drincw.Pathbuilder)
			return nil
		})

		handler.Stats.Log("finished", "took", handler.Stats.Diff(), "now", perf.Now())

		<-ctx.Done()

	}()
	return done
}

*/
