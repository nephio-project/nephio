/*
Copyright 2026 The Nephio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/integration"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/storage"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// RouterConfig holds configuration for the HTTP router
type RouterConfig struct {
	EnableCORS    bool
	EnableLogging bool
	EnableMetrics bool
	EnableHealth  bool
}

// DefaultRouterConfig returns a default router configuration
func DefaultRouterConfig() *RouterConfig {
	return &RouterConfig{
		EnableCORS:    true,
		EnableLogging: true,
		EnableMetrics: true,
		EnableHealth:  true,
	}
}

// SetupRouter configures and returns a Gin router with all handlers registered
func SetupRouter(storage storage.StorageInterface, config *RouterConfig, defaultNamespace string) *gin.Engine {
	return SetupRouterWithIntegration(storage, nil, config, defaultNamespace, false)
}

// SetupRouterWithIntegration configures and returns a Gin router with operator integration
func SetupRouterWithIntegration(storage storage.StorageInterface, operatorIntegration integration.OperatorIntegration, config *RouterConfig, defaultNamespace string, earlySchemaValidation bool) *gin.Engine {
	if config == nil {
		config = DefaultRouterConfig()
	}

	if defaultNamespace == "" {
		defaultNamespace = "focom-system"
	}

	// Set Gin mode to release for production
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Add middleware
	setupMiddleware(router, config)

	// Register health and metrics endpoints
	if config.EnableHealth {
		setupHealthEndpoints(router, storage)
	}

	if config.EnableMetrics {
		setupMetricsEndpoints(router)
	}

	// Create handlers with optional operator integration
	var oCloudHandler *OCloudHandler
	var templateInfoHandler *TemplateInfoHandler
	var fprHandler *FocomProvisioningRequestHandler

	if operatorIntegration != nil {
		oCloudHandler = NewOCloudHandlerWithIntegration(storage, operatorIntegration, defaultNamespace)
		templateInfoHandler = NewTemplateInfoHandlerWithIntegration(storage, operatorIntegration, defaultNamespace)
		fprHandler = NewFocomProvisioningRequestHandlerWithIntegration(storage, operatorIntegration, defaultNamespace, earlySchemaValidation)
	} else {
		oCloudHandler = NewOCloudHandler(storage, defaultNamespace)
		templateInfoHandler = NewTemplateInfoHandler(storage, defaultNamespace)
		fprHandler = NewFocomProvisioningRequestHandlerWithOptions(storage, defaultNamespace, earlySchemaValidation)
	}

	// Register routes at root level for backward compatibility
	oCloudHandler.RegisterRoutes(router)
	templateInfoHandler.RegisterRoutes(router)
	fprHandler.RegisterRoutes(router)

	// Also register under /api/v1 prefix to match OpenAPI spec
	apiV1 := router.Group("/api/v1")
	oCloudHandler.RegisterRoutes(apiV1)
	templateInfoHandler.RegisterRoutes(apiV1)
	fprHandler.RegisterRoutes(apiV1)

	return router
}

// setupMiddleware configures middleware for the router
func setupMiddleware(router *gin.Engine, config *RouterConfig) {
	// Recovery middleware
	router.Use(gin.Recovery())

	// Request logging middleware
	if config.EnableLogging {
		router.Use(requestLoggingMiddleware())
	}

	// CORS middleware
	if config.EnableCORS {
		router.Use(corsMiddleware())
	}

	// Request ID middleware
	router.Use(requestIDMiddleware())
}

// requestLoggingMiddleware creates a custom logging middleware
func requestLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request details
		logger := log.Log.WithName("nbi-http")

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("HTTP Request",
			"method", method,
			"path", path,
			"status", statusCode,
			"latency", latency,
			"clientIP", clientIP,
			"requestId", c.GetString("requestId"),
		)
	}
}

// requestIDMiddleware adds a unique request ID to each request
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate a simple request ID (in production, use a proper UUID library)
			requestID = generateRequestID()
		}

		c.Set("requestId", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// generateRequestID generates a simple request ID
func generateRequestID() string {
	// Simple timestamp-based ID for now
	// In production, use a proper UUID library
	return time.Now().Format("20060102150405.000000")
}

// corsMiddleware creates a simple CORS middleware
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Max-Age", "43200") // 12 hours

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// setupHealthEndpoints registers health check endpoints
func setupHealthEndpoints(router *gin.Engine, storage storage.StorageInterface) {
	health := router.Group("/health")
	{
		health.GET("/live", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":    "ok",
				"timestamp": time.Now(),
				"service":   "focom-nbi",
			})
		})

		health.GET("/ready", func(c *gin.Context) {
			// Check storage health
			if err := storage.HealthCheck(c.Request.Context()); err != nil {
				c.JSON(503, gin.H{
					"status":    "not ready",
					"timestamp": time.Now(),
					"service":   "focom-nbi",
					"error":     err.Error(),
				})
				return
			}

			c.JSON(200, gin.H{
				"status":    "ready",
				"timestamp": time.Now(),
				"service":   "focom-nbi",
			})
		})
	}
}

// setupMetricsEndpoints registers metrics endpoints
func setupMetricsEndpoints(router *gin.Engine) {
	metrics := router.Group("/metrics")
	{
		metrics.GET("", func(c *gin.Context) {
			// Basic metrics endpoint
			// In production, integrate with Prometheus metrics
			c.JSON(200, gin.H{
				"service":   "focom-nbi",
				"timestamp": time.Now(),
				"metrics": gin.H{
					"requests_total":   "counter_placeholder",
					"request_duration": "histogram_placeholder",
				},
			})
		})
	}
}

// APIInfo represents API information
type APIInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Timestamp   string `json:"timestamp"`
}

// setupAPIInfoEndpoint registers the API info endpoint
func SetupAPIInfoEndpoint(router *gin.Engine) {
	router.GET("/", func(c *gin.Context) {
		info := APIInfo{
			Name:        "FOCOM REST NBI",
			Version:     "v1alpha1",
			Description: "FOCOM North Bound Interface REST API",
			Timestamp:   time.Now().Format(time.RFC3339),
		}
		c.JSON(200, info)
	})

	router.GET("/api/info", func(c *gin.Context) {
		info := APIInfo{
			Name:        "FOCOM REST NBI",
			Version:     "v1alpha1",
			Description: "FOCOM North Bound Interface REST API",
			Timestamp:   time.Now().Format(time.RFC3339),
		}
		c.JSON(200, info)
	})
}
