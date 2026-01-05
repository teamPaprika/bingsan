package api

import (
	"fmt"
	"log/slog"
	"regexp"
	"sync"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kimuyb/bingsan/internal/api/handlers"
	"github.com/kimuyb/bingsan/internal/api/middleware"
	"github.com/kimuyb/bingsan/internal/config"
	"github.com/kimuyb/bingsan/internal/db"
	"github.com/kimuyb/bingsan/internal/events"
	_ "github.com/kimuyb/bingsan/internal/metrics" // Register custom Prometheus metrics
)

// polarisPathPattern matches /api/catalog/v1/{segment}/...
var polarisPathPattern = regexp.MustCompile(`^/api/catalog/v1/([^/]+)/(.*)$`)

var (
	promMiddlewareOnce sync.Once
	promMiddleware     *fiberprometheus.FiberPrometheus
)

// Server represents the HTTP server.
type Server struct {
	app    *fiber.App
	config *config.Config
	db     *db.DB
	events *events.Broker
}

// NewServer creates a new HTTP server.
func NewServer(cfg *config.Config, database *db.DB) *Server {
	app := fiber.New(fiber.Config{
		AppName:               "Iceberg REST Catalog",
		ServerHeader:          "iceberg-catalog",
		ReadTimeout:           cfg.Server.ReadTimeout,
		WriteTimeout:          cfg.Server.WriteTimeout,
		IdleTimeout:           cfg.Server.IdleTimeout,
		DisableStartupMessage: true,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
		ErrorHandler:          errorHandler,
		StrictRouting:         false, // /foo and /foo/ are the same
	})

	server := &Server{
		app:    app,
		config: cfg,
		db:     database,
		events: events.NewBroker(),
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

// EventBroker returns the event broker for publishing events.
func (s *Server) EventBroker() *events.Broker {
	return s.events
}

// App returns the underlying Fiber app for testing.
func (s *Server) App() *fiber.App {
	return s.app
}

// setupMiddleware configures global middleware.
func (s *Server) setupMiddleware() {
	// Request ID
	s.app.Use(requestid.New())

	// Recovery from panics
	s.app.Use(recover.New(recover.Config{
		EnableStackTrace: s.config.Server.Debug,
	}))

	// Prometheus metrics - use sync.Once to prevent duplicate registration in tests
	promMiddlewareOnce.Do(func() {
		promMiddleware = fiberprometheus.NewWithRegistry(prometheus.DefaultRegisterer.(*prometheus.Registry), "iceberg_catalog", "", "", nil)
		promMiddleware.SetSkipPaths([]string{"/health", "/ready", "/metrics"})
	})
	s.app.Use(promMiddleware.Middleware)
	// Use standard prometheus handler to expose all metrics (including custom ones)
	s.app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// CORS
	s.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,HEAD,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Iceberg-Access-Delegation",
	}))

	// Logging
	s.app.Use(middleware.Logger())

	// Polaris-compatible path rewriting (opt-in via config)
	// Rewrites /api/catalog/v1/{catalog}/... to /v1/...
	// This enables compatibility with polaris-tools benchmarks
	// Note: oauth paths are excluded from rewriting (oauth is not a catalog name)
	if s.config.Compat.PolarisEnabled {
		s.app.Use(func(c *fiber.Ctx) error {
			path := c.Path()
			if matches := polarisPathPattern.FindStringSubmatch(path); matches != nil {
				// matches[1] is the first segment after v1 (could be catalog name or oauth)
				// matches[2] is the rest of the path
				// Skip rewriting for oauth paths
				if matches[1] != "oauth" {
					c.Path("/v1/" + matches[2])
				}
			}
			return c.Next()
		})
	}
}

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	// Health check (no auth required)
	s.app.Get("/health", handlers.HealthCheck)
	s.app.Get("/ready", handlers.ReadyCheck(s.db))

	// WebSocket event stream (authentication via query param)
	s.app.Use("/v1/events", handlers.WebSocketUpgrade)
	s.app.Get("/v1/events/stream", handlers.EventStream(s.events, s.config, s.db))

	// API v1 routes
	v1 := s.app.Group("/v1")

	// Configuration endpoint
	v1.Get("/config", handlers.GetConfig(s.config))

	// OAuth2 endpoints (deprecated but required for compatibility)
	v1.Post("/oauth/tokens", handlers.ExchangeToken(s.config, s.db))

	// Polaris compatibility routes (opt-in via config)
	if s.config.Compat.PolarisEnabled {
		// Polaris-compatible OAuth endpoint (for polaris-tools benchmarks)
		s.app.Post("/api/catalog/v1/oauth/tokens", handlers.ExchangeToken(s.config, s.db))

		// Polaris Management API compatibility (for polaris-tools benchmarks)
		// These are mock endpoints since bingsan is a single-catalog system
		mgmt := s.app.Group("/api/management/v1")
		mgmt.Post("/catalogs", handlers.PolarisCreateCatalog)
		mgmt.Get("/catalogs", handlers.PolarisListCatalogs)
		mgmt.Get("/catalogs/:name", handlers.PolarisGetCatalog)
	}

	// Apply auth middleware if enabled
	if s.config.Auth.Enabled {
		v1.Use(middleware.Auth(s.config, s.db))
	}

	// Namespace routes
	v1.Get("/namespaces", handlers.ListNamespaces(s.db))
	v1.Post("/namespaces", handlers.CreateNamespace(s.db, s.events))
	v1.Get("/namespaces/:namespace", handlers.GetNamespace(s.db))
	v1.Head("/namespaces/:namespace", handlers.NamespaceExists(s.db))
	v1.Delete("/namespaces/:namespace", handlers.DeleteNamespace(s.db, s.events))
	v1.Post("/namespaces/:namespace/properties", handlers.UpdateNamespaceProperties(s.db))

	// Table routes
	v1.Get("/namespaces/:namespace/tables", handlers.ListTables(s.db))
	v1.Post("/namespaces/:namespace/tables", handlers.CreateTable(s.db, s.config))
	v1.Get("/namespaces/:namespace/tables/:table", handlers.LoadTable(s.db))
	v1.Post("/namespaces/:namespace/tables/:table", handlers.CommitTable(s.db, s.config))
	v1.Delete("/namespaces/:namespace/tables/:table", handlers.DropTable(s.db))
	v1.Head("/namespaces/:namespace/tables/:table", handlers.TableExists(s.db))
	v1.Get("/namespaces/:namespace/tables/:table/credentials", handlers.LoadCredentials(s.config))
	v1.Post("/namespaces/:namespace/tables/:table/metrics", handlers.ReportMetrics())

	// Scan planning routes (Iceberg REST spec compliant)
	v1.Post("/namespaces/:namespace/tables/:table/plan", handlers.SubmitScanPlan(s.db))
	v1.Get("/namespaces/:namespace/tables/:table/plan/:planId", handlers.GetScanPlan(s.db))
	v1.Delete("/namespaces/:namespace/tables/:table/plan/:planId", handlers.CancelScanPlan(s.db))
	v1.Post("/namespaces/:namespace/tables/:table/tasks", handlers.FetchPlanTasks(s.db))

	// Table registration
	v1.Post("/namespaces/:namespace/register", handlers.RegisterTable(s.db, s.config))

	// View routes
	v1.Get("/namespaces/:namespace/views", handlers.ListViews(s.db))
	v1.Post("/namespaces/:namespace/views", handlers.CreateView(s.db, s.config))
	v1.Get("/namespaces/:namespace/views/:view", handlers.LoadView(s.db))
	v1.Post("/namespaces/:namespace/views/:view", handlers.ReplaceView(s.db, s.config))
	v1.Delete("/namespaces/:namespace/views/:view", handlers.DropView(s.db))
	v1.Head("/namespaces/:namespace/views/:view", handlers.ViewExists(s.db))

	// Global table/view rename endpoints
	v1.Post("/tables/rename", handlers.RenameTable(s.db))
	v1.Post("/views/rename", handlers.RenameView(s.db))

	// Multi-table transactions
	v1.Post("/transactions/commit", handlers.CommitTransaction(s.db, s.config))

	// Also register with prefix for catalogs that use prefixes
	// e.g., /v1/my-catalog/namespaces
	prefix := v1.Group("/:prefix")
	prefix.Get("/namespaces", handlers.ListNamespaces(s.db))
	prefix.Post("/namespaces", handlers.CreateNamespace(s.db, s.events))
	prefix.Get("/namespaces/:namespace", handlers.GetNamespace(s.db))
	prefix.Head("/namespaces/:namespace", handlers.NamespaceExists(s.db))
	prefix.Delete("/namespaces/:namespace", handlers.DeleteNamespace(s.db, s.events))
	prefix.Post("/namespaces/:namespace/properties", handlers.UpdateNamespaceProperties(s.db))
	prefix.Get("/namespaces/:namespace/tables", handlers.ListTables(s.db))
	prefix.Post("/namespaces/:namespace/tables", handlers.CreateTable(s.db, s.config))
	prefix.Get("/namespaces/:namespace/tables/:table", handlers.LoadTable(s.db))
	prefix.Post("/namespaces/:namespace/tables/:table", handlers.CommitTable(s.db, s.config))
	prefix.Delete("/namespaces/:namespace/tables/:table", handlers.DropTable(s.db))
	prefix.Head("/namespaces/:namespace/tables/:table", handlers.TableExists(s.db))
	prefix.Get("/namespaces/:namespace/tables/:table/credentials", handlers.LoadCredentials(s.config))
	prefix.Post("/namespaces/:namespace/tables/:table/metrics", handlers.ReportMetrics())
	prefix.Post("/namespaces/:namespace/tables/:table/plan", handlers.SubmitScanPlan(s.db))
	prefix.Get("/namespaces/:namespace/tables/:table/plan/:planId", handlers.GetScanPlan(s.db))
	prefix.Delete("/namespaces/:namespace/tables/:table/plan/:planId", handlers.CancelScanPlan(s.db))
	prefix.Post("/namespaces/:namespace/tables/:table/tasks", handlers.FetchPlanTasks(s.db))
	prefix.Post("/namespaces/:namespace/register", handlers.RegisterTable(s.db, s.config))
	prefix.Get("/namespaces/:namespace/views", handlers.ListViews(s.db))
	prefix.Post("/namespaces/:namespace/views", handlers.CreateView(s.db, s.config))
	prefix.Get("/namespaces/:namespace/views/:view", handlers.LoadView(s.db))
	prefix.Post("/namespaces/:namespace/views/:view", handlers.ReplaceView(s.db, s.config))
	prefix.Delete("/namespaces/:namespace/views/:view", handlers.DropView(s.db))
	prefix.Head("/namespaces/:namespace/views/:view", handlers.ViewExists(s.db))
	prefix.Post("/tables/rename", handlers.RenameTable(s.db))
	prefix.Post("/views/rename", handlers.RenameView(s.db))
	prefix.Post("/transactions/commit", handlers.CommitTransaction(s.db, s.config))
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	slog.Info("server listening", "address", addr)
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	// Close event broker first
	if s.events != nil {
		s.events.Close()
	}
	return s.app.ShutdownWithTimeout(30 * time.Second)
}

// errorHandler handles errors globally.
func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	slog.Error("request error",
		"error", err,
		"status", code,
		"path", c.Path(),
		"method", c.Method(),
	)

	return c.Status(code).JSON(fiber.Map{
		"error": fiber.Map{
			"message": message,
			"type":    "ServerError",
			"code":    code,
		},
	})
}
