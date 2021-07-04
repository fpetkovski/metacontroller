package options

import (
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

type ConfigOption func(c *Configuration)

const (
	DefaultDiscoveryInterval      = 30 * time.Second
	DefaultInformerRelistInterval = 30 * time.Minute
	DefaultGoQPS                  = 5
	DefaultGoBurst                = 10
	DefaultWorkers                = 5
	DefaultEventsQPS              = 1. / 300
	DefaultEventsBurst            = 25
)

type Configuration struct {
	RestConfig        *rest.Config
	DiscoveryInterval time.Duration
	InformerRelist    time.Duration
	Workers           int
	CorrelatorOptions record.CorrelatorOptions
	MetricsEndpoint   string
}

func WithRestConfig(config *rest.Config) ConfigOption {
	return func(c *Configuration) {
		c.RestConfig = config
	}
}

func WithDiscoveryInterval(interval time.Duration) ConfigOption {
	return func(c *Configuration) {
		c.DiscoveryInterval = interval
	}
}
func WithInformerRelistInterval(interval time.Duration) ConfigOption {
	return func(c *Configuration) {
		c.InformerRelist = interval
	}
}

func WithNumberOfWorkers(workers int) ConfigOption {
	return func(c *Configuration) {
		c.Workers = workers
	}
}

func WithMetricsEndpoint(endpoint string) ConfigOption {
	return func(c *Configuration) {
		c.MetricsEndpoint = endpoint
	}
}

func WithCorrelatorOptions(correlatorOptions record.CorrelatorOptions) ConfigOption {
	return func(c *Configuration) {
		c.CorrelatorOptions = correlatorOptions
	}
}

func NewConfiguration(options ...ConfigOption) Configuration {
	c := Configuration{
		RestConfig:        nil,
		DiscoveryInterval: DefaultDiscoveryInterval,
		InformerRelist:    DefaultInformerRelistInterval,
		Workers:           DefaultWorkers,
		CorrelatorOptions: record.CorrelatorOptions{
			QPS:       DefaultEventsQPS,
			BurstSize: DefaultEventsBurst,
		},
		MetricsEndpoint: "0.0.0.0:9999/metrics",
	}

	for _, option := range options {
		option(&c)
	}

	return c
}
