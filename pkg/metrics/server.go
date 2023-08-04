package metrics

import (
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StartServer starts the HTTP server providing the metrics for scraping.
func StartServer(logger logr.Logger, port uint16) {
	go func() {
		handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		serveMux := http.NewServeMux()
		serveMux.Handle("/metrics", handler)

		for {
			err := http.ListenAndServe(fmt.Sprintf(":%d", port), serveMux)
			if err == http.ErrServerClosed {
				break
			}
			if err != nil {
				logger.Error(err, "Metrics server terminated unexpectedly and will be restarted")
			}
		}
	}()
}
