package catalog

import (
	"github.com/JulienBalestra/monitoring/pkg/collector"
	"github.com/JulienBalestra/monitoring/pkg/collector/bluetooth"
	"github.com/JulienBalestra/monitoring/pkg/collector/dnsmasq"
	"github.com/JulienBalestra/monitoring/pkg/collector/load"
	"github.com/JulienBalestra/monitoring/pkg/collector/lunar"
	"github.com/JulienBalestra/monitoring/pkg/collector/memory"
	"github.com/JulienBalestra/monitoring/pkg/collector/network"
	"github.com/JulienBalestra/monitoring/pkg/collector/tagger"
	"github.com/JulienBalestra/monitoring/pkg/collector/temperature"
	"github.com/JulienBalestra/monitoring/pkg/collector/wl"
)

func CollectorCatalog() map[string]func(*collector.Config) collector.Collector {
	return map[string]func(*collector.Config) collector.Collector{
		bluetooth.CollectorLoadName:          bluetooth.NewBluetooth,
		lunar.CollectorLoadName:              lunar.NewAcaia,
		dnsmasq.CollectorDnsMasqName:         dnsmasq.NewDnsMasq,
		dnsmasq.CollectorDnsMasqLogName:      dnsmasq.NewDnsMasqLog,
		load.CollectorLoadName:               load.NewLoad,
		memory.CollectorMemoryName:           memory.NewMemory,
		network.CollectorARPName:             network.NewARP,
		network.CollectorConntrackName:       network.NewConntrack,
		network.CollectorStatisticsName:      network.NewStatistics,
		network.CollectorWirelessName:        network.NewWireless,
		temperature.CollectorTemperatureName: temperature.NewTemperature,
		tagger.CollectorName:                 tagger.NewTagger,
		wl.CollectorWLName:                   wl.NewWL,
	}
}
