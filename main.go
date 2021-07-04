/*
Copyright 2017 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"metacontroller/pkg/options"
	"metacontroller/pkg/server"

	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2/klogr"
	controllerruntime "sigs.k8s.io/controller-runtime"
	controllerruntimelog "sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/klog/v2"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

var (
	discoveryInterval = flag.Duration("discovery-interval", options.DefaultDiscoveryInterval, "How often to refresh discovery cache to pick up newly-installed resources")
	informerRelist    = flag.Duration("cache-flush-interval", options.DefaultInformerRelistInterval, "How often to flush local caches and relist objects from the API server")
	metricsAddr       = flag.String("debug-addr", ":9999", "The address to bind metrics endpoint - /metrics")
	clientConfigPath  = flag.String("client-config-path", "", "(Deprecated: switch to `--kubeconfig`) Path to kubeconfig file (same format as used by kubectl); if not specified, use in-cluster config")
	clientGoQPS       = flag.Float64("client-go-qps", options.DefaultGoQPS, "Number of queries per second client-go is allowed to make (default 5)")
	clientGoBurst     = flag.Int("client-go-burst", options.DefaultGoBurst, "Allowed burst queries for client-go (default 10)")
	workers           = flag.Int("workers", options.DefaultWorkers, "Number of sync workers to run (default 5)")
	eventsQPS         = flag.Float64("events-qps", options.DefaultEventsQPS, "Rate of events flowing per object (default - 1 event per 5 minutes)")
	eventsBurst       = flag.Int("events-burst", options.DefaultEventsBurst, "Number of events allowed to send per object (default 25)")
	version           = "No version provided"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()
	// If client-config-path is set, overwrite the kubeconfig value
	// to maintain backward compatibility.
	if clientConfigPath != nil {
		err := flag.Set("kubeconfig", *clientConfigPath)
		if err != nil {
			klog.ErrorS(err, "Terminating")
			os.Exit(1)
		}
	}

	klog.InfoS("Discovery cache flush interval", "discovery_interval", *discoveryInterval)
	klog.InfoS("API server object cache flush interval", "cache_flush_interval", *informerRelist)
	klog.InfoS("Metrics http server address", "port", *metricsAddr)
	klog.InfoS("Metacontroller build information", "version", version)

	logger := klogr.NewWithOptions(klogr.WithFormat(klogr.FormatKlog))
	controllerruntimelog.SetLogger(logger)
	config, err := controllerruntime.GetConfig()
	if err != nil {
		klog.ErrorS(err, "Terminating")
		os.Exit(1)
	}
	config.QPS = float32(*clientGoQPS)
	config.Burst = *clientGoBurst

	configuration := options.Configuration{
		RestConfig:        config,
		DiscoveryInterval: *discoveryInterval,
		InformerRelist:    *informerRelist,
		Workers:           *workers,
		CorrelatorOptions: record.CorrelatorOptions{
			BurstSize: *eventsBurst,
			QPS:       float32(*eventsQPS),
		},
		MetricsEndpoint: *metricsAddr,
	}

	// Create a new manager with a stop function
	// for resource cleanup
	mgr, stopManager, err := server.New(configuration)
	if err != nil {
		klog.ErrorS(err, "Terminating")
		os.Exit(1)
	}

	// Use a WaitGroup to make sure the metrics server
	// and controller manager stop gracefully
	// before we exit
	var wg sync.WaitGroup
	wg.Add(1)
	mgrStopChan := signals.SetupSignalHandler()
	go func() {
		defer wg.Done()
		if err := mgr.Start(mgrStopChan); err != nil {
			klog.ErrorS(err, "Terminating")
			os.Exit(1)
		}
		stopManager()
		klog.InfoS("Stopped metacontroller")
	}()

	<-mgrStopChan.Done()
	klog.InfoS("Stopped controller manager")
	wg.Wait()
}
