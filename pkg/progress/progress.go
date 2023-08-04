// Package progress provides Reader and Writer
package progress

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

// Reader consistently writes the number of bytes read to Progress.
type Reader struct {
	io.Reader       // Reader to read from
	Bytes     int64 // total number of bytes read (so far)

	Rewritable
}

func (cr *Reader) Read(bytes []byte) (int, error) {
	count, err := cr.Reader.Read(bytes)
	cr.Bytes += int64(count)
	cr.Rewritable.Write(fmt.Sprintf("Read %s", humanize.Bytes(uint64(cr.Bytes))))
	return count, err
}

// Writer consistently writes the number of bytes written to Progress.
type Writer struct {
	io.Writer       // Writer to write to
	Bytes     int64 // Total number of bytes written

	Rewritable
}

func (cw *Writer) Write(bytes []byte) (int, error) {
	cw.Bytes += int64(len(bytes))
	cw.Rewritable.Write(fmt.Sprintf("Wrote %s", humanize.Bytes(uint64(cw.Bytes))))
	return cw.Writer.Write(bytes)
}

// DefaultFlushInterval is a reasonable default flush interval
const DefaultFlushInterval = time.Second / 30

type Rewritable struct {
	Writer io.Writer

	FlushInterval  time.Duration // minimum time between flushes of the progress
	lastFlush      time.Time     // last time we flushed
	longestContent int           // longest content ever flushed
	content        string        // current content
}

func (rw *Rewritable) Write(value string) {
	rw.content = value
	rw.Flush(false)
}

func (rw *Rewritable) Flush(force bool) {
	if !(force || time.Since(rw.lastFlush) > rw.FlushInterval) {
		return
	}

	// determine the longest string we ever flushed to the output
	if len(rw.content) >= rw.longestContent {
		rw.longestContent = len(rw.content)
	}

	// add a blanking space behind the content
	blank := strings.Repeat(" ", rw.longestContent-len(rw.content))
	fmt.Fprintf(rw.Writer, "\r%s%s", rw.content, blank)

	// and flush the actual data
	rw.lastFlush = time.Now()
}

func (rw *Rewritable) Close() {
	rw.content = ""
	rw.Flush(true)
	rw.Writer.Write([]byte("\r"))
}

type Progress struct {
	Rewritable
}

func (progress *Progress) Set(prefix string, count, total int) {
	totalS := strconv.Itoa(total)
	countS := strconv.Itoa(count)
	if len(countS) < len(totalS) {
		countS = strings.Repeat(" ", len(totalS)-len(countS)) + countS
	}

	if countS < totalS {
		progress.Rewritable.Write(fmt.Sprintf("%s: %s/%s", prefix, countS, totalS))
	} else {
		progress.Rewritable.Write(fmt.Sprintf("%s: %s", prefix, countS))
	}
}
