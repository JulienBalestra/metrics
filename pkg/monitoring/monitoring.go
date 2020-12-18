package monitoring

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/JulienBalestra/dry/pkg/version"
	"github.com/JulienBalestra/monitoring/pkg/collector"
	"github.com/JulienBalestra/monitoring/pkg/collector/catalog"
	"github.com/JulienBalestra/monitoring/pkg/datadog"
	"github.com/JulienBalestra/monitoring/pkg/datadog/forward"
	"github.com/JulienBalestra/monitoring/pkg/tagger"
	"go.uber.org/zap"
)

func NewDefaultConfig() *Config {
	zapConfig := zap.NewProductionConfig()
	return &Config{
		DatadogClientConfig: &datadog.Config{
			ClientMetrics: &datadog.ClientMetrics{},
		},
		ZapConfig: &zapConfig,
	}
}

type Config struct {
	HostTags   []string
	ConfigFile string
	Hostname   string
	ZapConfig  *zap.Config
	ZapLevel   string

	DatadogClientConfig *datadog.Config
}

type Monitoring struct {
	conf *Config

	datadogClient *datadog.Client
	catalogConfig *catalog.ConfigFile

	Tagger *tagger.Tagger
}

func NewMonitoring(conf *Config) (*Monitoring, error) {
	if conf.Hostname == "" {
		return nil, fmt.Errorf("empty hostname")
	}
	if conf.DatadogClientConfig.SendInterval <= datadog.MinimalSendInterval {
		return nil, fmt.Errorf("SendInterval must be greater or equal to %s", datadog.MinimalSendInterval)
	}
	_, err := tagger.CreateTags(conf.HostTags...)
	if err != nil {
		return nil, err
	}

	catalogConfig, err := catalog.ParseConfigFile(conf.ConfigFile)
	if err != nil {
		return nil, err
	}

	datadogClient := datadog.NewClient(conf.DatadogClientConfig)
	err = conf.ZapConfig.Level.UnmarshalText([]byte(conf.ZapLevel))
	if err != nil {
		return nil, err
	}
	err = zap.RegisterSink(forward.DatadogZapScheme, forward.NewDatadogForwarder(context.Background(), datadogClient))
	if err != nil {
		return nil, err
	}
	logger, err := conf.ZapConfig.Build()
	if err != nil {
		return nil, err
	}
	logger = logger.With(zap.Int("pid", os.Getpid()))
	zap.ReplaceGlobals(logger)
	zap.RedirectStdLog(logger)
	return &Monitoring{
		conf:          conf,
		datadogClient: datadogClient,
		catalogConfig: catalogConfig,
		Tagger:        tagger.NewTagger(),
	}, nil
}

func (m *Monitoring) Start(ctx context.Context) error {

	runCtx, runCancel := context.WithCancel(ctx)
	waitGroup := &sync.WaitGroup{}

	waitGroup.Add(1)
	go func() {
		m.datadogClient.Run(runCtx)
		waitGroup.Done()
	}()
	defer close(m.datadogClient.ChanSeries)

	errorsChan := make(chan error, len(m.catalogConfig.Collectors))
	defer close(errorsChan)

	for name, newFn := range catalog.CollectorCatalog() {
		select {
		case <-runCtx.Done():
			break
		default:
			nb := 0
			zctx := zap.L().With(
				zap.String("collector", name),
			)
			for _, collectorToStart := range m.catalogConfig.Collectors {
				if collectorToStart.Name != name {
					continue
				}
				nb++
				config := &collector.Config{
					MetricsClient:   m.datadogClient,
					Tagger:          m.Tagger,
					Host:            m.conf.Hostname,
					CollectInterval: collectorToStart.Interval,
					Options:         collectorToStart.Options,
				}
				c := newFn(config)
				waitGroup.Add(1)
				go func(coll collector.Collector) {
					errorsChan <- collector.RunCollection(runCtx, coll)
					waitGroup.Done()
				}(c)
			}
			if nb == 0 {
				zctx.Info("ignoring collector")
				continue
			}
			zctx.Info("collector started", zap.Int("instances", nb))
		}
	}
	tags := append(m.Tagger.GetUnstable(m.conf.Hostname),
		"commit:"+version.Commit[:8],
	)
	m.datadogClient.MetricClientUp(m.conf.Hostname, tags...)
	_ = m.datadogClient.UpdateHostTags(runCtx, m.conf.HostTags)
	select {
	case <-runCtx.Done():
	case err := <-errorsChan:
		zap.L().Error("failed to run collection", zap.Error(err))
		runCancel()
	}

	ctxShutdown, runCancel := context.WithTimeout(context.Background(), time.Second*5)
	_ = m.datadogClient.MetricClientShutdown(ctxShutdown, m.conf.Hostname, tags...)
	runCancel()
	waitGroup.Wait()
	return nil
}