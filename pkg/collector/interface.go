package collector

import (
	"context"
	"time"

	"github.com/JulienBalestra/monitoring/pkg/metrics"
	"github.com/JulienBalestra/monitoring/pkg/tagger"
	"go.uber.org/zap"
)

type Config struct {
	SeriesCh chan metrics.Series
	Tagger   *tagger.Tagger

	Host            string
	CollectInterval time.Duration
}

func (c Config) OverrideCollectInterval(d time.Duration) *Config {
	c.CollectInterval = d
	return &c
}

type Collector interface {
	Config() *Config
	Collect(context.Context) error
	Name() string
	IsDaemon() bool
}

func RunCollection(ctx context.Context, c Collector) error {
	config := c.Config()

	zctx := zap.L().With(
		zap.String("collector", c.Name()),
		zap.Duration("collectionInterval", config.CollectInterval),
	)

	if c.IsDaemon() {
		zctx.Info("collecting metrics continuously")
		err := c.Collect(ctx)
		if err != nil {
			return err
		}
		return nil
	}

	ticker := time.NewTicker(config.CollectInterval)
	defer ticker.Stop()
	zctx.Info("collecting metrics periodically")

	for {
		select {
		case <-ctx.Done():
			zctx.Info("end of collection")
			return ctx.Err()

		case <-ticker.C:
			err := c.Collect(ctx)
			if err != nil {
				zctx.Error("failed collection", zap.Error(err))
				continue
			}
			zctx.Info("successfully run collection")
		}
	}
}
