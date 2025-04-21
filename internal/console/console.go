package console

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/tkw1536/pkglib/status"
)

// New creates a new console.
func New(scrollback int) *Console {
	if scrollback <= 0 {
		scrollback = 1000
	}

	console := Console{
		ScrollBack: scrollback,

		buffer: &status.LineBuffer{},
		lines:  make([]string, 0, scrollback),
	}

	console.grid = widget.NewTextGrid()
	console.scroll = container.NewScroll(console.grid)

	console.buffer.Line = console.line

	return &console
}

// Console represents a buffer that can be written to and is displayed in a monospaced scroll container.
type Console struct {
	// ScrollBack is the maximum number of lines stored by the container
	ScrollBack int
	buffer     *status.LineBuffer
	lines      []string

	scroll *container.Scroll
	grid   *widget.TextGrid
}

// CanvasObject returns a CanvasObject that can be used to render this console.
func (console *Console) CanvasObject() fyne.CanvasObject {
	return console.scroll
}

// Write writes to the underlying console.
func (console *Console) Write(data []byte) (int, error) {
	count, err := console.buffer.Write(data)
	if err != nil {
		return count, fmt.Errorf("failed to write to buffer: %w", err)
	}
	return count, nil
}

func (console *Console) line(line string) {
	if len(console.lines) < console.ScrollBack {
		console.lines = append(console.lines, line)
	} else {
		// size was shrunk, resize the console size array
		if len(console.lines) > console.ScrollBack {
			old := console.lines
			console.lines = make([]string, console.ScrollBack)
			copy(console.lines, old[len(old)-console.ScrollBack:])
		}

		// copy over the tail
		copy(console.lines, console.lines[1:])
		console.lines[console.ScrollBack-1] = line
	}

	// and scroll to the bottom
	console.grid.SetText(strings.Join(console.lines, "\n"))
	console.scroll.ScrollToBottom()
}
