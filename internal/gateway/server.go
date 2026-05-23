// Package gateway wires the HTTP routes, middleware, and handlers.
package gateway

import (
	"log/slog"
	"net/http"
	"time"

	chi "github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/yawo/onefacture/internal/adapters/registry"
	"github.com/yawo/onefacture/internal/config"
	"github.com/yawo/onefacture/internal/directory"
	"github.com/yawo/onefacture/internal/events"
	"github.com/yawo/onefacture/internal/gateway/middleware"
	"github.com/yawo/onefacture/internal/gateway/openapi"
	"github.com/yawo/onefacture/internal/gateway/routes"
	"github.com/yawo/onefacture/internal/storage"
	"github.com/yawo/onefacture/internal/validation"
)

// Server holds the assembled router and its dependencies.
type Server struct {
	opts Options
	r    *chi.Mux
}

// Options groups the dependencies needed to build the gateway.
type Options struct {
	Config    *config.Config
	Logger    *slog.Logger
	Store     *storage.Store
	Validator *validation.Client
	Registry  *registry.Registry
	Events    *events.Bus
	AuthN     *middleware.APIKeyAuth
}

// New wires the dependencies into a Server.
func New(opts Options) *Server {
	s := &Server{opts: opts}
	s.r = s.buildRouter()
	return s
}

// Router returns the HTTP handler.
func (s *Server) Router() http.Handler { return s.r }

func (s *Server) buildRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(middleware.RequestIDHeader)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(middleware.AccessLog(s.opts.Logger))
	r.Use(middleware.Metrics)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-API-Key", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	auth := s.opts.AuthN.WithPepper(s.opts.Config.Auth.HashPepper)

	deps := routes.Dependencies{
		Logger:     s.opts.Logger,
		Store:      s.opts.Store,
		Validator:  s.opts.Validator,
		Registry:   s.opts.Registry,
		HashPepper: s.opts.Config.Auth.HashPepper,
		Directory: directory.NewResolver(5*time.Minute,
			directory.StaticProvider{Source: "static", Entries: map[string]string{}},
			directory.FallbackProvider{PAID: "mock", Source: "fallback"},
		),
		Events: s.opts.Events,
	}

	// Public endpoints.
	r.Get("/healthz", routes.Health)
	r.Get("/readyz", routes.Ready(s.opts.Store, s.opts.Events))
	r.Get("/metrics", promhttp.Handler().ServeHTTP)
	r.Get("/docs", openapi.ScalarHandler(s.opts.Config.HTTP.PublicBaseURL))
	r.Get("/tools/compliance-dashboard", routes.ComplianceDashboardUI)
	r.Get("/tools/webhook-inspector", routes.WebhookInspectorUI)
	r.Get("/openapi.json", openapi.SpecHandler())
	r.Get("/openapi.yaml", openapi.SpecHandler())
	r.Get("/v1/platforms", routes.ListPlatforms(deps))
	r.Post("/v1/sandbox/credentials", routes.CreateSandboxCredentials(deps))

	// Authenticated v1.
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware)
		rl := middleware.NewRateLimit(s.opts.Events.Client(), s.opts.Config.HTTP.RateLimitPerMin)
		r.Use(rl.Middleware)

		r.Route("/v1/invoices", func(r chi.Router) {
			r.Post("/", routes.CreateInvoice(deps))
			r.Get("/", routes.ListInvoices(deps))
			r.Get("/rejections/summary", routes.RejectionSummary(deps))
			r.Get("/{id}", routes.GetInvoice(deps))
			r.Post("/{id}/submit", routes.SubmitInvoice(deps))
			r.Post("/{id}/retry", routes.RetryRejectedInvoice(deps))
			r.Get("/{id}/rejection-patch", routes.RejectionPatch(deps))
			r.Get("/{id}/events", routes.InvoiceEvents(deps))
			r.Get("/{id}/timeline", routes.InvoiceTimeline(deps))
		})

		r.Route("/v1/inbox", func(r chi.Router) {
			r.Get("/", routes.ListInbox(deps))
			r.Post("/{id}/approve", routes.ApproveInbox(deps))
		})

		r.Post("/v1/validate", routes.ValidateRaw(deps))
		r.Post("/v1/validate/bulk", routes.ValidateBulk(deps))
		r.Get("/v1/directory/lookup", routes.DirectoryLookup(deps))
		r.Get("/v1/submissions/dlq", routes.ListSubmissionDLQ(deps))
		r.Post("/v1/submissions/dlq/{id}/replay", routes.ReplaySubmissionDLQ(deps))

		r.Route("/v1/webhooks", func(r chi.Router) {
			r.Post("/", routes.CreateWebhook(deps))
			r.Get("/deliveries", routes.ListWebhookDeliveries(deps))
			r.Post("/deliveries/{id}/replay", routes.ReplayWebhookDelivery(deps))
		})

		// GDPR endpoints
		r.Post("/v1/data/export", routes.GDPRExport(deps))
		r.Post("/v1/data/erase", routes.GDPRErase(deps))
		r.Get("/v1/analytics/compliance-score", routes.ComplianceScore(deps))
		r.Get("/v1/analytics/rejection-retry-success-rate", routes.RejectionRetrySuccessRate(deps))
	})

	return r
}
