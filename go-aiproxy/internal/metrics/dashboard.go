package metrics

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*
var templatesFS embed.FS

// DashboardConfig holds configuration for the metrics dashboard
type DashboardConfig struct {
	Title           string
	RefreshInterval int // seconds
	PrometheusURL   string
}

// MetricsDashboard provides a web-based metrics dashboard
type MetricsDashboard struct {
	config *DashboardConfig
	tmpl   *template.Template
}

// NewDashboard creates a new metrics dashboard
func NewDashboard(config *DashboardConfig) (*MetricsDashboard, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &MetricsDashboard{
		config: config,
		tmpl:   tmpl,
	}, nil
}

// Handler returns a Gin handler for the dashboard
func (d *MetricsDashboard) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"title":           d.config.Title,
			"refreshInterval": d.config.RefreshInterval,
			"prometheusURL":   d.config.PrometheusURL,
		})
	}
}

// RegisterRoutes registers dashboard routes
func (d *MetricsDashboard) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/dashboard", d.Handler())
	router.Static("/static", "./static")
}