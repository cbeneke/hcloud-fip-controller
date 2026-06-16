package fipcontroller

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics emitted by the controller. They are registered on the
// default registry and served on the /metrics endpoint of the health server.
var (
	reconcileTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fip_controller_reconciliations_total",
		Help: "Total number of reconciliation runs by result.",
	}, []string{"result"})

	reconcileDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "fip_controller_reconcile_duration_seconds",
		Help:    "Duration of reconciliation runs in seconds.",
		Buckets: prometheus.DefBuckets,
	})

	reassignmentsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "fip_controller_floating_ip_reassignments_total",
		Help: "Total number of floating IP (re)assignments performed.",
	})

	managedFloatingIPs = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "fip_controller_managed_floating_ips",
		Help: "Number of floating IPs currently managed by the controller.",
	})

	leaderGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "fip_controller_leader",
		Help: "Whether this instance is the elected leader (1) or not (0).",
	})
)
