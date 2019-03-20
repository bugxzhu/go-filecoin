package metrics

import (
	"gx/ipfs/QmNVpHFt7QmabuVQyguf8AbkLDZoFh7ifBYztqijYT1Sd2/go.opencensus.io/stats"
	"gx/ipfs/QmNVpHFt7QmabuVQyguf8AbkLDZoFh7ifBYztqijYT1Sd2/go.opencensus.io/stats/view"
)

// Opencensus stats
var (
	MApplyMessageMs = stats.Float64("consensus/apply_message", "The duration in milliseconds of ApplyMessage", stats.UnitMilliseconds)

	applyMessageView = &view.View{
		Name:        "consensus/apply_message",
		Measure:     MApplyMessageMs,
		Description: "The distribution of the durations",

		// Latency in buckets:
		// [>=0ms, >=25ms, >=50ms, >=75ms, >=100ms, >=200ms, >=400ms, >=600ms, >=800ms, >=1s, >=2s, >=4s, >=8s]
		Aggregation: view.Distribution(25, 50, 75, 100, 200, 400, 600, 800, 1000, 2000, 4000, 8000),
	}
)
