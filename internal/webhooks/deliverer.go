// Package webhooks signs and delivers outbound webhook events.
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yawo/onefacture/internal/events"
	"github.com/yawo/onefacture/internal/storage"
)

// Deliverer subscribes to the event bus, signs payloads with the per-endpoint
// secret, and POSTs them to the registered URLs with retry/backoff.
type Deliverer struct {
	logger *slog.Logger
	bus    *events.Bus
	store  *storage.Store
	client *http.Client
}

func NewDeliverer(logger *slog.Logger, bus *events.Bus, store *storage.Store) *Deliverer {
	return &Deliverer{
		logger: logger,
		bus:    bus,
		store:  store,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Run blocks until the context is cancelled.
func (d *Deliverer) Run(ctx context.Context) {
	go d.dispatchLoop(ctx)
	if err := d.bus.Subscribe(ctx, "webhooks", "deliverer-1", d.onEvent); err != nil {
		if ctx.Err() == nil {
			d.logger.Error("webhooks subscribe", "err", err)
		}
	}
}

func (d *Deliverer) onEvent(ctx context.Context, ev events.Event) error {
	orgID, err := uuid.Parse(ev.OrganizationID)
	if err != nil {
		return nil
	}
	endpoints, err := d.store.Webhooks.ListActive(ctx, orgID, ev.Type)
	if err != nil {
		d.logger.Warn("list webhooks", "err", err)
		return err
	}
	for _, ep := range endpoints {
		_, err := d.store.Webhooks.Enqueue(ctx, ep.ID, ev.Type, map[string]any{
			"type":            ev.Type,
			"occurred_at":     ev.OccurredAt,
			"organization_id": ev.OrganizationID,
			"invoice_id":      ev.InvoiceID,
			"data":            ev.Payload,
		})
		if err != nil {
			d.logger.Warn("enqueue delivery", "err", err)
		}
	}
	return nil
}

func (d *Deliverer) dispatchLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.flushOnce(ctx)
		}
	}
}

func (d *Deliverer) flushOnce(ctx context.Context) {
	due, err := d.store.Webhooks.NextDue(ctx, 20)
	if err != nil {
		d.logger.Warn("webhooks NextDue", "err", err)
		return
	}
	for _, delivery := range due {
		d.attempt(ctx, delivery)
	}
}

func (d *Deliverer) attempt(ctx context.Context, delivery storage.WebhookDelivery) {
	ep, err := d.store.Webhooks.GetEndpoint(ctx, delivery.EndpointID)
	if err != nil {
		d.logger.Warn("endpoint lookup", "err", err)
		return
	}
	payload, err := json.Marshal(delivery.Payload)
	if err != nil {
		d.logger.Warn("payload marshal", "err", err)
		return
	}
	if err := endpointAllowed(ctx, ep); err != nil {
		attempts := delivery.Attempts + 1
		_ = d.store.Webhooks.MarkFailed(ctx, delivery.ID, attempts, backoff(attempts), err.Error())
		return
	}
	sig := sign(ep.SecretHash, payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, bytes.NewReader(payload))
	if err != nil {
		d.logger.Warn("new request", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Onefacture-Event", delivery.EventType)
	req.Header.Set("X-Onefacture-Signature", "sha256="+sig)
	req.Header.Set("X-Onefacture-Delivery", delivery.ID.String())

	client := d.client
	if ep.MTLSRequired {
		client, err = d.clientForEndpoint(ep)
		if err != nil {
			attempts := delivery.Attempts + 1
			_ = d.store.Webhooks.MarkFailed(ctx, delivery.ID, attempts, backoff(attempts), err.Error())
			return
		}
	}
	resp, err := client.Do(req)
	attempts := delivery.Attempts + 1
	if err != nil {
		_ = d.store.Webhooks.MarkFailed(ctx, delivery.ID, attempts, backoff(attempts), err.Error())
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_ = d.store.Webhooks.MarkDelivered(ctx, delivery.ID)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	_ = d.store.Webhooks.MarkFailed(ctx, delivery.ID, attempts, backoff(attempts),
		fmt.Sprintf("status=%d body=%s", resp.StatusCode, string(body)))
}

func (d *Deliverer) clientForEndpoint(ep *storage.WebhookEndpoint) (*http.Client, error) {
	certPath, keyPath, err := parseMTLSCertRef(ep.MTLSCertRef)
	if err != nil {
		return nil, err
	}
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read mtls cert: %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read mtls key: %w", err)
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("load mtls key pair: %w", err)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	return &http.Client{Timeout: d.client.Timeout, Transport: transport}, nil
}

func parseMTLSCertRef(ref string) (string, string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", fmt.Errorf("mtls_cert_ref required when mtls_required is true")
	}
	parts := strings.Split(ref, ":")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("mtls_cert_ref must be cert_path:key_path")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func endpointAllowed(ctx context.Context, ep *storage.WebhookEndpoint) error {
	if len(ep.IPAllowlist) == 0 {
		return nil
	}
	parsed, err := url.Parse(ep.URL)
	if err != nil {
		return fmt.Errorf("parse webhook url: %w", err)
	}
	host := parsed.Hostname()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve webhook host: %w", err)
	}
	allowed := map[string]struct{}{}
	for _, ip := range ep.IPAllowlist {
		allowed[ip] = struct{}{}
	}
	for _, ip := range ips {
		if _, ok := allowed[ip.IP.String()]; ok {
			return nil
		}
	}
	return fmt.Errorf("webhook destination %s not in ip_allowlist", host)
}

func sign(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func backoff(attempts int) time.Time {
	// Exponential backoff capped at 1 hour.
	d := time.Duration(1<<attempts) * time.Second
	if d > time.Hour {
		d = time.Hour
	}
	return time.Now().UTC().Add(d)
}
