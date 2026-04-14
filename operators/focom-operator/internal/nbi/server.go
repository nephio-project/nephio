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

package nbi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/handlers"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/integration"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/storage"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Server represents the HTTP server for the NBI REST API
type Server struct {
	port    int
	router  *gin.Engine
	server  *http.Server
	storage storage.StorageInterface
}

// ServerConfig holds configuration for the NBI server
type ServerConfig struct {
	Port                  int
	RouterConfig          *handlers.RouterConfig
	StorageConfig         interface{}                     // For future storage configuration
	OperatorIntegration   integration.OperatorIntegration // Optional operator integration
	DefaultNamespace      string                          // Default namespace for resources
	EarlySchemaValidation bool                            // Enable schema validation on create/update
}

// DefaultServerConfig returns a default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:             8080,
		RouterConfig:     handlers.DefaultRouterConfig(),
		DefaultNamespace: "focom-system",
	}
}

// NewServer creates a new NBI HTTP server
func NewServer(storage storage.StorageInterface, config *ServerConfig) *Server {
	if config == nil {
		config = DefaultServerConfig()
	}

	// Setup router with all handlers and middleware
	router := handlers.SetupRouterWithIntegration(storage, config.OperatorIntegration, config.RouterConfig, config.DefaultNamespace, config.EarlySchemaValidation)

	// Setup API info endpoint
	handlers.SetupAPIInfoEndpoint(router)

	return &Server{
		port:    config.Port,
		router:  router,
		storage: storage,
	}
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second, // Increased for long-running operations like CreateDraftFromRevision
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("Starting NBI HTTP server", "port", s.port)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(err, "Failed to start HTTP server")
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	logger := log.FromContext(ctx)

	if s.server == nil {
		return nil
	}

	logger.Info("Stopping NBI HTTP server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}

// GetRouter returns the gin router for registering handlers
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}
