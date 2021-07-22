package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    readyGauge = prometheus.NewGauge(
    	prometheus.GaugeOpts{
    		Name: "kubevirt_cnao_ready",
    		Help: "Operator Components (Deployed and)? Ready",
    	})
)

func StartPrometheus() error {
	return startPrometheusEndpoint()
}

// startPrometheusEndpoint starts an http server providing a prometheus endpoint
func startPrometheusEndpoint() error {
	initGauges()

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

func initGauges() {
	prometheus.MustRegister(readyGauge)
}

func SetReadyGauge(isReady bool) {
	if isReady {
		readyGauge.Set(1)
	} else {
		readyGauge.Set(0)
	}
}
