package metrics

import (
	"context"
	"net/http"
	"time"

	ma "gx/ipfs/QmNTCey11oxhb1AxDnQBRHtdhap6Ctud872NjAYPYYXPuc/go-multiaddr"
	manet "gx/ipfs/QmZcLBXKaFe8ND5YHPkJRAwmhJGrVsi1JqDZNyJ4nRK5Mj/go-multiaddr-net"

	"gx/ipfs/QmNVpHFt7QmabuVQyguf8AbkLDZoFh7ifBYztqijYT1Sd2/go.opencensus.io/exporter/prometheus"
	"gx/ipfs/QmNVpHFt7QmabuVQyguf8AbkLDZoFh7ifBYztqijYT1Sd2/go.opencensus.io/stats"
	"gx/ipfs/QmNVpHFt7QmabuVQyguf8AbkLDZoFh7ifBYztqijYT1Sd2/go.opencensus.io/stats/view"
	"gx/ipfs/QmNVpHFt7QmabuVQyguf8AbkLDZoFh7ifBYztqijYT1Sd2/go.opencensus.io/zpages"
	prom "gx/ipfs/QmaQtvgBNGwD4os5VLWtBLR6HM6TY6ApX6xFqSnfjDF2aW/client_golang/prometheus"

	"github.com/filecoin-project/go-filecoin/config"
)

// NewRegistry creates a new registry
func NewRegistry() *Registry {
	return &Registry{
		measures: make(map[string]*Measurement),
	}
}

// Registry holds a list of measurements
type Registry struct {
	measures map[string]*Measurement
}

// Measurement contains a opencensus measurement
type Measurement struct {
	measure   *stats.Float64Measure
	view      *view.View
	startTime time.Time
	Record    func(ctx context.Context)
}

// NewTimer creates a float64 measurement that can be recorded via the Record function
func (r *Registry) NewTimer(name, desc, unit string) *Measurement {
	// TODO add locking
	msur, ok := r.measures[name]
	if !ok {
		fMeasure := stats.Float64(name, desc, unit)
		fView := &view.View{
			Name:        name,
			Measure:     fMeasure,
			Description: desc,
			// [>=0ms, >=25ms, >=50ms, >=75ms, >=100ms, >=200ms, >=400ms, >=600ms, >=800ms, >=1s, >=2s, >=4s, >=8s]
			Aggregation: view.Distribution(25, 50, 75, 100, 200, 400, 600, 800, 1000, 2000, 4000, 8000),
		}
		if err := view.Register(fView); err != nil {
			// yes I know this is a bad idea, only doing it for prototype
			panic(err)
		}

		fStart := time.Now()
		msur := &Measurement{
			measure:   fMeasure,
			view:      fView,
			startTime: fStart,
			Record: func(ctx context.Context) {
				stats.Record(ctx, fMeasure.M(float64(time.Since(fStart).Round(time.Millisecond))/1e6))
			},
		}

		r.measures[name] = msur
		return msur
	}

	fStart := time.Now()
	return &Measurement{
		measure:   msur.measure,
		view:      msur.view,
		startTime: fStart,
		Record: func(ctx context.Context) {
			stats.Record(ctx, msur.measure.M(float64(time.Since(fStart).Round(time.Millisecond))/1e6))
		},
	}
}

// SetupMetrics registers and serves prometheus metrics
func SetupMetrics(cfg *config.MetricsConfig) error {
	if !cfg.PrometheusEnabled {
		return nil
	}

	// validate config values and marshal to types
	interval, err := time.ParseDuration(cfg.ReportInterval)
	if err != nil {
		log.Errorf("invalid metrics interval: %s", err)
		return err
	}

	promma, err := ma.NewMultiaddr(cfg.PrometheusEndpoint)
	if err != nil {
		return err
	}

	_, promAddr, err := manet.DialArgs(promma)
	if err != nil {
		return err
	}

	// setup prometheus
	registry := prom.NewRegistry()
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "filecoin",
		Registry:  registry,
	})
	if err != nil {
		return err
	}

	view.RegisterExporter(pe)
	view.SetReportingPeriod(interval)
	if err := view.Register(applyMessageView); err != nil {
		return err
	}

	go func() {
		mux := http.NewServeMux()
		zpages.Handle(mux, "/debug")
		mux.Handle("/metrics", pe)
		if err := http.ListenAndServe(promAddr, mux); err != nil {
			log.Errorf("failed to serve /metrics endpoint on %v", err)
		}
	}()

	return nil
}
