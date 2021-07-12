package monitoring

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartPrometheus() error {
	return startPrometheusEndpoint()
}

// startPrometheusEndpoint starts an http server providing a prometheus endpoint
func startPrometheusEndpoint() error {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		server := http.Server{
			Addr:    "0.0.0.0:8443",
			Handler: mux,
		}
		log.Printf("Starting Prometheus metrics endpoint server")
		err := server.ListenAndServe()
		if err != nil {
			log.Fatalf("Failed to start Prometheus metrics endpoint server: %v", err)
		}
	}()
	return nil
}
