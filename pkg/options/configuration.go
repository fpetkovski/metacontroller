package options

import (
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

type ConfigOption func(c *Configuration)

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
		DiscoveryInterval: 30 * time.Second,
		InformerRelist:    30 * time.Minute,
		Workers:           5,
		CorrelatorOptions: record.CorrelatorOptions{
			QPS:       1. / 300.,
			BurstSize: 25,
		},
		MetricsEndpoint: "0.0.0.0:9999/metrics",
	}

	for _, option := range options {
		option(&c)
	}

	return c
}
