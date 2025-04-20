package headache

import (
	"fmt"
	"math"
	"math/big"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/FAU-CDI/hangover/internal/stats"
)

func newStatus() status {
	return status{
		done: binding.NewBool(),

		stage: binding.NewString(),
		bar:   widget.NewProgressBar(),

		directTriples:     binding.NewString(),
		datumTriples:      binding.NewString(),
		maskedPredTriples: binding.NewString(),
		maskedDataTriples: binding.NewString(),
		inverseTriples:    binding.NewString(),
		conflictTriples:   binding.NewString(),
	}
}

type status struct {
	// are we done?
	done binding.Bool

	stage binding.String
	bar   *widget.ProgressBar

	// triples counter
	directTriples     binding.String
	datumTriples      binding.String
	maskedPredTriples binding.String
	maskedDataTriples binding.String
	inverseTriples    binding.String
	conflictTriples   binding.String
}

func (status *status) Set(stats *stats.Stats) {
	_ = status.done.Set(stats.Done())

	// setup index stats
	istats := stats.IndexStats()
	setUint64(status.directTriples, istats.DirectTriples)
	setUint64(status.datumTriples, istats.DatumTriples)
	setUint64(status.maskedPredTriples, istats.MaskedPredTriples)
	setUint64(status.maskedDataTriples, istats.MaskedDataTriples)
	setUint64(status.inverseTriples, istats.InverseTriples)
	setUint64(status.conflictTriples, istats.ConflictTriples)

	// setup current stats
	current := stats.Current()
	_ = status.stage.Set(string(current.Stage))

	if current.Total != 0 && current.Current <= current.Total {
		text := fmt.Sprintf("%d/%d", current.Current, current.Total)
		status.bar.TextFormatter = func() string { return text }
		percent, _ := big.NewFloat(0).Quo(
			big.NewFloat(0).SetInt64(int64(current.Current)),
			big.NewFloat(0).SetInt64(int64(current.Total)),
		).Float64()
		status.bar.Value = percent
	} else {
		status.bar.TextFormatter = func() string { return "" }
		status.bar.Value = math.Inf(1)
	}

	if status.bar.Visible() {
		fyne.Do(status.bar.Refresh)
	}
}

func setUint64(binding binding.String, value uint64) {
	_ = binding.Set(strconv.FormatUint(value, 10))
}
