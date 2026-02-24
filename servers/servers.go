package servers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Prometheus_server(latestSweepTxHash *prometheus.GaugeVec, latestRefundTxHash *prometheus.GaugeVec) {
	reg := prometheus.NewRegistry()

	reg.MustRegister(
		latestSweepTxHash,
		latestRefundTxHash,
	)

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("Server is running"))
		if err != nil {
			fmt.Printf("Error writing response: %s \n", err)
		}
	})

	log.Println("Starting prometheus server on :2555")
	if err := http.ListenAndServe(":2555", nil); err != nil {
		log.Fatalf("Error starting server: %s", err)
	}
}
