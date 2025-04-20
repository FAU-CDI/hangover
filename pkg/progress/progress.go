// Package progress provides Reader and Writer
package progress

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

// Reader consistently writes the number of bytes read to Progress.
type Reader struct {
	io.Reader // Reader to read from
	Rewritable
	Bytes int64 // total number of bytes read (so far)
}

var errReaderOverflow = errors.New("Writer.Write: bytes overflow")

func (cr *Reader) Read(bytes []byte) (int, error) {
	count, err := cr.Reader.Read(bytes)
	cr.Bytes += int64(count)
	if err != nil {
		return count, fmt.Errorf("failed to read: %w", err)
	}
	byteCount := cr.Bytes
	if byteCount < 0 {
		return 0, errReaderOverflow
	}
	if err := cr.Write("Read " + humanize.Bytes(uint64(byteCount))); err != nil {
		return count, fmt.Errorf("failed to write to rewritable: %w", err)
	}
	return count, err
}

// Writer consistently writes the number of bytes written to Progress.
type Writer struct {
	io.Writer // Writer to write to
	Rewritable
	Bytes int64 // Total number of bytes written
}

var errWriterOverflow = errors.New("Writer.Write: bytes overflow")

func (cw *Writer) Write(bytes []byte) (int, error) {
	cw.Bytes += int64(len(bytes))

	byteCount := cw.Bytes
	if byteCount < 0 {
		return 0, errWriterOverflow
	}
	if err := cw.Rewritable.Write("Wrote " + humanize.Bytes(uint64(byteCount))); err != nil {
		return 0, fmt.Errorf("failed to write to Rewritable: %w", err)
	}
	v, err := cw.Writer.Write(bytes)
	if err != nil {
		return v, fmt.Errorf("failed to write to Writer: %w", err)
	}
	return v, nil
}

// DefaultFlushInterval is a reasonable default flush interval.
const DefaultFlushInterval = time.Second / 30

type Rewritable struct {
	lastFlush      time.Time // last time we flushed
	Writer         io.Writer
	content        string        // current content
	FlushInterval  time.Duration // minimum time between flushes of the progress
	longestContent int           // longest content ever flushed
}

func (rw *Rewritable) Write(value string) error {
	rw.content = value
	err := rw.Flush(false)
	if err != nil {
		return fmt.Errorf("failed to flush: %w", err)
	}
	return nil
}

func (rw *Rewritable) Flush(force bool) error {
	if !force && time.Since(rw.lastFlush) <= rw.FlushInterval {
		return nil
	}

	// determine the longest string we ever flushed to the output
	if len(rw.content) >= rw.longestContent {
		rw.longestContent = len(rw.content)
	}

	// add a blanking space behind the content
	blank := strings.Repeat(" ", rw.longestContent-len(rw.content))
	_, err := fmt.Fprintf(rw.Writer, "\r%s%s", rw.content, blank)
	if err != nil {
		return fmt.Errorf("failed to flush: %w", err)
	}

	// and flush the actual data
	rw.lastFlush = time.Now()

	return nil
}

// Close resets any output written to the terminal.
// After a call to Close(), further calls to Set may re-use it.
func (rw *Rewritable) Close() error {
	rw.content = ""
	if err := rw.Flush(true); err != nil {
		return fmt.Errorf("failed to flush: %w", err)
	}

	if _, err := rw.Writer.Write([]byte("\r")); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}
	rw.longestContent = 0
	return nil
}

type Progress struct {
	Rewritable
}

func (progress *Progress) Set(prefix string, count, total int) error {
	totalS := strconv.Itoa(total)
	countS := strconv.Itoa(count)
	if len(countS) < len(totalS) {
		countS = strings.Repeat(" ", len(totalS)-len(countS)) + countS
	}

	var err error
	if countS < totalS {
		err = progress.Write(fmt.Sprintf("%s: %s/%s", prefix, countS, totalS))
	} else {
		err = progress.Write(fmt.Sprintf("%s: %s", prefix, countS))
	}
	if err != nil {
		return fmt.Errorf("failed to write to progress: %w", err)
	}
	return nil
}
