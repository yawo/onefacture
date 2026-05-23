package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// PA submissions
	PASubmissionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "onefacture_pa_submission_total",
			Help: "Total number of submissions to Plateformes Agréées",
		},
		[]string{"pa_id", "status"}, // accepted, rejected, error, etc.
	)

	PASubmissionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "onefacture_submission_duration_seconds",
			Help:    "Duration of PA submission operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"pa_id"},
	)

	// DLQ
	DLQDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "onefacture_dlq_depth",
			Help: "Current number of messages in the submission DLQ",
		},
	)

	DLQEnqueuedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "onefacture_dlq_enqueued_total",
			Help: "Total messages enqueued to DLQ",
		},
		[]string{"pa_id"},
	)

	// Webhooks
	WebhookDeliveryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "onefacture_webhook_delivery_total",
			Help: "Total webhook delivery attempts",
		},
		[]string{"status"}, // success, failed, retry
	)

	// Compliance
	ComplianceScore7d = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "onefacture_compliance_score_7d",
			Help: "Rolling 7-day compliance score (0-100)",
		},
	)

	// HTTP
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// Adapter calls (goes through reliability wrapper)
	AdapterCallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "onefacture_adapter_calls_total",
			Help: "Total calls to PA adapters",
		},
		[]string{"pa_id", "operation", "status"}, // operation: submit, get_status, etc.
	)
)
