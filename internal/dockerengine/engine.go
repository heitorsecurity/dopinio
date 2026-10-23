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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AppDeployment represents the deployment specification for a Dopinio app on Docker
type AppDeployment struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Image       string            `json:"image"`
	Domain      string            `json:"domain"`
	Port        int               `json:"port"`
	Replicas    int               `json:"replicas"`
	Environment map[string]string `json:"environment"`
	Labels      map[string]string `json:"labels"`
	Volumes     []string          `json:"volumes"`
}

// AppStatus represents the runtime status of an application running in Docker
type AppStatus struct {
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Status      string    `json:"status"` // "running", "stopped", "deploying", "failed"
	ContainerID string    `json:"container_id"`
	Image       string    `json:"image"`
	URL         string    `json:"url"`
	Port        int       `json:"port"`
	CreatedAt   time.Time `json:"created_at"`
	Restarts    int       `json:"restarts"`
}

// EngineConfig holds configuration for connecting to Docker daemon
type EngineConfig struct {
	Host          string
	NetworkName   string
	DefaultDomain string
	RegistryURL   string
}

// DockerEngine is the interface for orchestrating applications directly on Docker
type DockerEngine interface {
	DeployApp(ctx context.Context, app AppDeployment) (*AppStatus, error)
	GetApp(ctx context.Context, namespace, appName string) (*AppStatus, error)
	ListApps(ctx context.Context, namespace string) ([]AppStatus, error)
	DeleteApp(ctx context.Context, namespace, appName string) error
	ScaleApp(ctx context.Context, namespace, appName string, replicas int) error
	StreamLogs(ctx context.Context, namespace, appName string, follow bool, out io.Writer) error
	EnsureNetwork(ctx context.Context) error
}

// DefaultDockerEngine implements DockerEngine using Docker HTTP REST / socket API
type DefaultDockerEngine struct {
	config     EngineConfig
	socketPath string
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewDockerEngine creates a new DockerEngine instance
func NewDockerEngine(cfg EngineConfig) (*DefaultDockerEngine, error) {
	if cfg.NetworkName == "" {
		cfg.NetworkName = "dopinio-net"
	}
	if cfg.DefaultDomain == "" {
		cfg.DefaultDomain = "dopinio.localhost"
	}

	socketPath := cfg.Host
	if socketPath == "" {
		if os.Getenv("DOCKER_HOST") != "" {
			socketPath = os.Getenv("DOCKER_HOST")
		} else {
			socketPath = "/var/run/docker.sock"
		}
	}

	return &DefaultDockerEngine{
		config:     cfg,
		socketPath: socketPath,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// ContainerLabels creates Traefik routing and Dopinio identification labels
func (e *DefaultDockerEngine) ContainerLabels(app AppDeployment) map[string]string {
	labels := map[string]string{
		"dopinio.managed":    "true",
		"dopinio.namespace":  app.Namespace,
		"dopinio.app":        app.Name,
		"traefik.enable":     "true",
	}

	hostDomain := app.Domain
	if hostDomain == "" {
		hostDomain = fmt.Sprintf("%s.%s.%s", app.Name, app.Namespace, e.config.DefaultDomain)
	}

	routerName := fmt.Sprintf("dopinio-%s-%s", app.Namespace, app.Name)
	labels[fmt.Sprintf("traefik.http.routers.%s.rule", routerName)] = fmt.Sprintf("Host(`%s`)", hostDomain)
	labels[fmt.Sprintf("traefik.http.routers.%s.entrypoints", routerName)] = "web,websecure"

	appPort := app.Port
	if appPort <= 0 {
		appPort = 8080
	}
	labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", routerName)] = fmt.Sprintf("%d", appPort)

	for k, v := range app.Labels {
		labels[k] = v
	}

	return labels
}

// ContainerName builds a deterministic container name
func (e *DefaultDockerEngine) ContainerName(namespace, appName string, instance int) string {
	cleanNamespace := strings.ToLower(strings.ReplaceAll(namespace, "_", "-"))
	cleanAppName := strings.ToLower(strings.ReplaceAll(appName, "_", "-"))
	if instance <= 1 {
		return fmt.Sprintf("dopinio-%s-%s", cleanNamespace, cleanAppName)
	}
	return fmt.Sprintf("dopinio-%s-%s-%d", cleanNamespace, cleanAppName, instance)
}

