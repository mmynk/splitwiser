package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/mmynk/splitwiser/internal/storage/sqlite"
)

// flyNetworkOnly restricts the handler to Fly.io's private IPv6 network (fdaa::/7).
// The managed Prometheus scraper (fly-metrics.net) runs inside this network,
// so the Grafana dashboard continues to work. All external requests return 403.
// In local dev (APP_ENV != production), all requests are allowed.
func flyNetworkOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("APP_ENV") == "production" {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if !strings.HasPrefix(ip, "fdaa") {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// splitwiserCollector implements prometheus.Collector to expose DB-level gauges.
// Queries run on each scrape, so counts are always current without background goroutines.
type splitwiserCollector struct {
	store  *sqlite.SQLiteStore
	users  *prometheus.Desc
	bills  *prometheus.Desc
	groups *prometheus.Desc
}

func newCollector(store *sqlite.SQLiteStore) *splitwiserCollector {
	return &splitwiserCollector{
		store:  store,
		users:  prometheus.NewDesc("splitwiser_users_total", "Total registered users.", nil, nil),
		bills:  prometheus.NewDesc("splitwiser_bills_total", "Total bills in the database.", nil, nil),
		groups: prometheus.NewDesc("splitwiser_groups_total", "Total groups in the database.", nil, nil),
	}
}

func (c *splitwiserCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.users
	ch <- c.bills
	ch <- c.groups
}

func (c *splitwiserCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := c.store.GetStats(context.Background())
	if err != nil {
		slog.Warn("metrics: failed to get stats", "error", err)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.users, prometheus.GaugeValue, float64(stats.Users))
	ch <- prometheus.MustNewConstMetric(c.bills, prometheus.GaugeValue, float64(stats.Bills))
	ch <- prometheus.MustNewConstMetric(c.groups, prometheus.GaugeValue, float64(stats.Groups))
}
