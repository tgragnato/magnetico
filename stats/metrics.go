package stats

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	instance *Stats
	once     sync.Once
)

const namespace = "magnetico"

// GetStats returns a singleton instance of Stats
func GetInstance() *Stats {
	once.Do(func() {
		instance = &Stats{
			bootstrap: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "bootstrap",
				Help:      "Number of times the bootstrap process has been triggered",
			}),
			writeError: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "write_error",
				Help:      "Number of times there was an error writing a message to the UDP socket",
			}),
			readError: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "read_error",
				Help:      "Number of times there was an error reading a message from the UDP socket",
			}),
			rtClearing: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rt_clearing",
				Help:      "Number of times the routing table has been cleared",
			}),
			nonUTF8: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "non_utf8",
				Help:      "Number of times a torrent has been ignored due to its name not being UTF-8 compliant",
			}),
			checkError: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "check_error",
				Help:      "Number of times there was an error checking whether a torrent exists",
			}),
			addError: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "add_error",
				Help:      "Number of times there was an error adding a torrent to the database",
			}),
			mseEncryption: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "mse_encryption",
				Help:      "Number of times a peer connection has been obfuscated with MSE",
			}),
			extensions: map[string]prometheus.Counter{},
		}
	})
	return instance
}

// Create a new registry for Prometheus using the Go and Process collectors
func MakePrometheusHandler() http.HandlerFunc {

	registry := prometheus.NewRegistry()

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{
			Namespace:    namespace,
			ReportErrors: false,
		}),
		GetInstance(),
	)

	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP
}
