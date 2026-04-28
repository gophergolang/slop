package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/vibeguard/platform/db"
	"github.com/vibeguard/platform/events"
	"github.com/vibeguard/team-task-saas/internal/auth"
	"github.com/vibeguard/team-task-saas/internal/tasks"
	"github.com/vibeguard/team-task-saas/pkg/middleware"
)

func main() {
	ctx := context.Background()
	logger, _ := zap.NewProduction()

	// Platform SDK clients (from declaration: multi-tenancy + event-driven)
	dbURL := os.Getenv("DATABASE_URL")
	database, err := db.NewPostgres(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	eventClient, err := events.NewClient(natsURL, logger)
	if err != nil {
		slog.Warn("NATS not available, using no-op publisher", "error", err)
		// In production, this would be fatal or use a fallback
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.StructuredLogger())
	router.Use(middleware.TenantExtractor()) // extracts from JWT or X-Tenant-ID header
	router.Use(middleware.RateLimiter("300/minute"))

	// Auth module (from declaration)
	authHandler := auth.NewHandler(pool)
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)
	}

	// Tasks module (strictly from declaration - only whitelisted operations)
	// Now using Platform SDK (db + events)
	taskHandler := tasks.NewHandler(database, eventClient)
	taskGroup := router.Group("/api/v1/tasks")
	taskGroup.Use(middleware.AuthRequired())
	taskGroup.Use(middleware.RolesAllowed("owner", "admin", "member"))
	{
		taskGroup.POST("", taskHandler.Create)                    // create: true
		taskGroup.GET("/:id", taskHandler.Get)                    // read: true
		taskGroup.GET("", taskHandler.List)                       // list: true
		taskGroup.PATCH("/:id", taskHandler.Update)               // update: [title, description, status, priority, due_date, assignee_id]
		// DELETE is NOT generated because delete: false in declaration

		// Custom endpoint from declaration
		taskGroup.POST("/:id/prioritize", taskHandler.Prioritize) // AI prioritization
	}

	// Teams module (from declaration)
	// ... similar pattern

	// Billing module (read-only from declaration)
	// ... 

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		slog.Info("server starting", "port", 8080)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}