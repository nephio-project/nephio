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
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error     string    `json:"error"`
	Code      string    `json:"code"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"requestId,omitempty"`
}

// BaseHandler provides common functionality for all handlers
type BaseHandler struct{}

// NewBaseHandler creates a new base handler
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}

// SendError sends a standardized error response
func (h *BaseHandler) SendError(c *gin.Context, statusCode int, errorCode, message, details string) {
	response := ErrorResponse{
		Error:     message,
		Code:      errorCode,
		Details:   details,
		Timestamp: time.Now(),
		RequestID: c.GetString("requestId"),
	}

	c.JSON(statusCode, response)
}

// SendSuccess sends a successful response with data
func (h *BaseHandler) SendSuccess(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// SendCreated sends a 201 Created response
func (h *BaseHandler) SendCreated(c *gin.Context, data interface{}) {
	h.SendSuccess(c, http.StatusCreated, data)
}

// SendOK sends a 200 OK response
func (h *BaseHandler) SendOK(c *gin.Context, data interface{}) {
	h.SendSuccess(c, http.StatusOK, data)
}

// SendAccepted sends a 202 Accepted response
func (h *BaseHandler) SendAccepted(c *gin.Context, data interface{}) {
	h.SendSuccess(c, http.StatusAccepted, data)
}

// SendNoContent sends a 204 No Content response
func (h *BaseHandler) SendNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// SendBadRequest sends a 400 Bad Request error
func (h *BaseHandler) SendBadRequest(c *gin.Context, code, message, details string) {
	h.SendError(c, http.StatusBadRequest, code, message, details)
}

// SendNotFound sends a 404 Not Found error
func (h *BaseHandler) SendNotFound(c *gin.Context, code, message, details string) {
	h.SendError(c, http.StatusNotFound, code, message, details)
}

// SendConflict sends a 409 Conflict error
func (h *BaseHandler) SendConflict(c *gin.Context, code, message, details string) {
	h.SendError(c, http.StatusConflict, code, message, details)
}

// SendInternalError sends a 500 Internal Server Error
func (h *BaseHandler) SendInternalError(c *gin.Context, code, message, details string) {
	h.SendError(c, http.StatusInternalServerError, code, message, details)
}
