package main

import (
	"strings"

	"fyne.io/fyne/v2/widget"
	"github.com/tkw1536/pkglib/status"
)

func NewWriterGrid(w, h int) *WriterGrid {
	wg := &WriterGrid{
		tg:    widget.NewTextGrid(),
		w:     w,
		h:     h,
		lines: make([]string, 0, h),
	}
	wg.lb.Line = wg.onWriteLine
	return wg
}

type WriterGrid struct {
	w, h int
	tg   *widget.TextGrid

	lb    status.LineBuffer
	lines []string
}

func (wg *WriterGrid) Write(data []byte) (int, error) {
	return wg.lb.Write(data)
}

func (wg *WriterGrid) toLine(line string) string {
	if len(line) < wg.w {
		return line + strings.Repeat(" ", wg.w-len(line))
	}
	if len(line) > wg.w {
		return line[:wg.w]
	}
	return line
}

func (wg *WriterGrid) onWriteLine(line string) {
	line = wg.toLine(line)

	if len(wg.lines) < wg.h {
		wg.lines = append(wg.lines, line)
	} else {
		copy(wg.lines[0:], wg.lines[1:])
		wg.lines[wg.h-1] = line
	}

	text := strings.Join(wg.lines, "\n")
	if len(wg.lines) < wg.h {
		text += ""
	}
	wg.tg.SetText(text)
}
