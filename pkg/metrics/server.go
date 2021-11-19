package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	klog "k8s.io/klog/v2"
)

// StartServer starts the HTTP server providing the metrics for scraping.
func StartServer(port uint16) {
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
				klog.ErrorS(err, "metrics server terminated unexpectedly and will be restarted")
			}
		}
	}()
}
