package collector

import (
    "context"
    "sync"
    "time"

    "github.com/kubestack-ai/kubestack-ai/internal/monitor/storage"
    "github.com/kubestack-ai/kubestack-ai/internal/monitor/model"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
    // Collect collects metrics and returns data points
    Collect(ctx context.Context) ([]*model.MetricPoint, error)

    // Name returns the collector name
    Name() string

    // Interval returns the collection interval
    Interval() time.Duration
}

// CollectorScheduler scheduler for collectors
type CollectorScheduler struct {
    collectors []MetricsCollector
    store      storage.TimeseriesStore
    stopCh     chan struct{}
    wg         sync.WaitGroup
	log        logger.Logger
}

// NewCollectorScheduler creates a new scheduler
func NewCollectorScheduler(store storage.TimeseriesStore, log logger.Logger) *CollectorScheduler {
    return &CollectorScheduler{
        collectors: make([]MetricsCollector, 0),
        store:      store,
        stopCh:     make(chan struct{}),
		log:        log,
    }
}

// Register registers a collector
func (s *CollectorScheduler) Register(collector MetricsCollector) {
    s.collectors = append(s.collectors, collector)
}

// Start starts the scheduler
func (s *CollectorScheduler) Start(ctx context.Context) error {
    for _, collector := range s.collectors {
        s.wg.Add(1)
        go s.runCollector(ctx, collector)
    }
    return nil
}

// runCollector runs a single collector loop
func (s *CollectorScheduler) runCollector(ctx context.Context, collector MetricsCollector) {
    defer s.wg.Done()
    ticker := time.NewTicker(collector.Interval())
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-s.stopCh:
            return
        case <-ticker.C:
            // Collect metrics
            points, err := collector.Collect(ctx)
            if err != nil {
                s.log.Errorf("[%s] Collection failed: %v", collector.Name(), err)
                continue
            }

            // Write to storage
            if err := s.store.Write(ctx, points); err != nil {
                s.log.Errorf("[%s] Storage failed: %v", collector.Name(), err)
            } else {
                s.log.Debugf("[%s] Collected %d metrics", collector.Name(), len(points))
            }
        }
    }
}

// Stop stops the scheduler
func (s *CollectorScheduler) Stop() {
    close(s.stopCh)
    s.wg.Wait()
}
