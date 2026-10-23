// Copyright © 2026 Dopinio Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dockerengine

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Router registers all Docker-native PaaS routes into Gin
type Router struct {
	engine DockerEngine
}

// NewRouter creates a new docker API router
func NewRouter(engine DockerEngine) *Router {
	return &Router{engine: engine}
}

// RegisterRoutes registers the simplified, clean REST endpoints for Dopinio apps
func (r *Router) RegisterRoutes(rg *gin.RouterGroup) {
	apps := rg.Group("/docker/namespaces/:namespace/apps")
	{
		apps.GET("", r.ListApps)
		apps.POST("", r.DeployApp)
		apps.GET("/:app", r.GetApp)
		apps.DELETE("/:app", r.DeleteApp)
		apps.POST("/:app/scale", r.ScaleApp)
		apps.GET("/:app/logs", r.StreamLogs)
	}

	// System health & readiness
	rg.GET("/docker/health", r.Health)
}

// Health checks if Docker Engine is accessible
func (r *Router) Health(c *gin.Context) {
	ctx := c.Request.Context()
	if err := r.engine.EnsureNetwork(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": fmt.Sprintf("Docker engine unavailable: %v", err),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"runtime": "docker",
	})
}

// ListApps returns all running applications in the given namespace
func (r *Router) ListApps(c *gin.Context) {
	ns := c.Param("namespace")
	ctx := c.Request.Context()

	apps, err := r.engine.ListApps(ctx, ns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"namespace": ns,
		"apps":      apps,
		"total":     len(apps),
	})
}

// DeployApp deploys or updates an application container on Docker
func (r *Router) DeployApp(c *gin.Context) {
	ns := c.Param("namespace")
	ctx := c.Request.Context()

	var req AppDeployment
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid payload: %v", err)})
		return
	}
	req.Namespace = ns

	status, err := r.engine.DeployApp(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Application deployed successfully",
		"app":     status,
	})
}

// GetApp returns details of an application
func (r *Router) GetApp(c *gin.Context) {
	ns := c.Param("namespace")
	appName := c.Param("app")
	ctx := c.Request.Context()

	status, err := r.engine.GetApp(ctx, ns, appName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// DeleteApp stops and removes an application container
func (r *Router) DeleteApp(c *gin.Context) {
	ns := c.Param("namespace")
	appName := c.Param("app")
	ctx := c.Request.Context()

	if err := r.engine.DeleteApp(ctx, ns, appName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("App %s/%s deleted successfully", ns, appName),
	})
}

// ScaleApp adjusts replica count
func (r *Router) ScaleApp(c *gin.Context) {
	ns := c.Param("namespace")
	appName := c.Param("app")
	ctx := c.Request.Context()

	replicasStr := c.DefaultQuery("replicas", "1")
	replicas, _ := strconv.Atoi(replicasStr)
	if replicas < 0 {
		replicas = 0
	}

	if err := r.engine.ScaleApp(ctx, ns, appName, replicas); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  fmt.Sprintf("App %s scaled to %d replicas", appName, replicas),
		"replicas": replicas,
	})
}

// StreamLogs streams application container logs
func (r *Router) StreamLogs(c *gin.Context) {
	ns := c.Param("namespace")
	appName := c.Param("app")
	follow := c.DefaultQuery("follow", "false") == "true"
	ctx := c.Request.Context()

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("X-Content-Type-Options", "nosniff")

	_ = r.engine.StreamLogs(ctx, ns, appName, follow, c.Writer)
}
