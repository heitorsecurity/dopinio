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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// DockerRawClient handles HTTP communication directly over Docker daemon socket/npipe/tcp
type DockerRawClient struct {
	client     *http.Client
	hostScheme string
	hostAddr   string
}

// NewDockerRawClient configures a transport for the docker socket/pipe
func NewDockerRawClient(host string) *DockerRawClient {
	if host == "" {
		host = "/var/run/docker.sock"
	}

	transport := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}

	scheme := "http"
	addr := host

	if strings.HasPrefix(host, "unix://") || strings.HasPrefix(host, "/") {
		socketPath := strings.TrimPrefix(host, "unix://")
		transport.DialContext = func(ctx context.Context, proto, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		}
		addr = "localhost"
	} else if strings.HasPrefix(host, "tcp://") {
		addr = strings.TrimPrefix(host, "tcp://")
	}

	return &DockerRawClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   120 * time.Second,
		},
		hostScheme: scheme,
		hostAddr:   addr,
	}
}

// Request performs a request against Docker Engine API
func (c *DockerRawClient) Request(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s://%s/v1.45%s", c.hostScheme, c.hostAddr, path)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.client.Do(req)
}

// EnsureNetwork ensures the bridge network exists
func (e *DefaultDockerEngine) EnsureNetwork(ctx context.Context) error {
	client := NewDockerRawClient(e.socketPath)
	networkName := e.config.NetworkName

	// Check if network exists
	resp, err := client.Request(ctx, http.MethodGet, fmt.Sprintf("/networks/%s", networkName), nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		_ = resp.Body.Close()
		return nil
	}
	if resp != nil {
		_ = resp.Body.Close()
	}

	// Create network
	createBody := map[string]interface{}{
		"Name":           networkName,
		"Driver":         "bridge",
		"CheckDuplicate": true,
		"Labels": map[string]string{
			"dopinio.managed": "true",
		},
	}
	jsonBody, err := json.Marshal(createBody)
	if err != nil {
		return err
	}

	resp, err = client.Request(ctx, http.MethodPost, "/networks/create", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create network %s: %w", networkName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create docker network (%d): %s", resp.StatusCode, string(respBytes))
	}

	return nil
}

// DeployApp deploys a container running on Docker with Traefik routing
func (e *DefaultDockerEngine) DeployApp(ctx context.Context, app AppDeployment) (*AppStatus, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	client := NewDockerRawClient(e.socketPath)

	if err := e.EnsureNetwork(ctx); err != nil {
		return nil, err
	}

	containerName := e.ContainerName(app.Namespace, app.Name, 1)

	// Stop & remove existing container if running
	_ = e.stopAndRemoveContainer(ctx, client, containerName)

	// Create env slice
	envList := make([]string, 0, len(app.Environment)+4)
	for k, v := range app.Environment {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}
	appPort := app.Port
	if appPort <= 0 {
		appPort = 8080
	}
	envList = append(envList, fmt.Sprintf("PORT=%d", appPort))
	envList = append(envList, fmt.Sprintf("DOPINIO_APP=%s", app.Name))
	envList = append(envList, fmt.Sprintf("DOPINIO_NAMESPACE=%s", app.Namespace))

	labels := e.ContainerLabels(app)

	// Build create container request
	createReq := map[string]interface{}{
		"Image":  app.Image,
		"Env":    envList,
		"Labels": labels,
		"HostConfig": map[string]interface{}{
			"NetworkMode":   e.config.NetworkName,
			"RestartPolicy": map[string]interface{}{"Name": "unless-stopped"},
			"Binds":         app.Volumes,
		},
	}

	jsonBytes, err := json.Marshal(createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal container config: %w", err)
	}

	createURL := fmt.Sprintf("/containers/create?name=%s", containerName)
	resp, err := client.Request(ctx, http.MethodPost, createURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create container %s: %w", containerName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		out, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docker create returned status %d: %s", resp.StatusCode, string(out))
	}

	var createResult struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
		return nil, fmt.Errorf("failed to decode create response: %w", err)
	}

	// Start container
	startURL := fmt.Sprintf("/containers/%s/start", createResult.ID)
	startResp, err := client.Request(ctx, http.MethodPost, startURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start container %s: %w", containerName, err)
	}
	defer startResp.Body.Close()

	if startResp.StatusCode != http.StatusNoContent && startResp.StatusCode != http.StatusOK {
		out, _ := io.ReadAll(startResp.Body)
		return nil, fmt.Errorf("docker start returned %d: %s", startResp.StatusCode, string(out))
	}

	hostDomain := app.Domain
	if hostDomain == "" {
		hostDomain = fmt.Sprintf("%s.%s.%s", app.Name, app.Namespace, e.config.DefaultDomain)
	}

	return &AppStatus{
		Name:        app.Name,
		Namespace:   app.Namespace,
		Status:      "running",
		ContainerID: createResult.ID,
		Image:       app.Image,
		URL:         fmt.Sprintf("http://%s", hostDomain),
		Port:        appPort,
		CreatedAt:   time.Now(),
	}, nil
}

// stopAndRemoveContainer gracefully stops and removes a container
func (e *DefaultDockerEngine) stopAndRemoveContainer(ctx context.Context, client *DockerRawClient, name string) error {
	stopURL := fmt.Sprintf("/containers/%s/stop?t=5", name)
	resp, _ := client.Request(ctx, http.MethodPost, stopURL, nil)
	if resp != nil {
		_ = resp.Body.Close()
	}

	delURL := fmt.Sprintf("/containers/%s?force=true", name)
	delResp, _ := client.Request(ctx, http.MethodDelete, delURL, nil)
	if delResp != nil {
		_ = delResp.Body.Close()
	}
	return nil
}

// GetApp inspects an app container
func (e *DefaultDockerEngine) GetApp(ctx context.Context, namespace, appName string) (*AppStatus, error) {
	client := NewDockerRawClient(e.socketPath)
	containerName := e.ContainerName(namespace, appName, 1)

	resp, err := client.Request(ctx, http.MethodGet, fmt.Sprintf("/containers/%s/json", containerName), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("app %s/%s not found", namespace, appName)
	}

	var inspect struct {
		ID      string `json:"Id"`
		Created string `json:"Created"`
		State   struct {
			Status    string `json:"Status"`
			RestartCount int `json:"RestartCount"`
		} `json:"State"`
		Config struct {
			Image  string            `json:"Image"`
			Labels map[string]string `json:"Labels"`
		} `json:"Config"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&inspect); err != nil {
		return nil, err
	}

	hostDomain := fmt.Sprintf("%s.%s.%s", appName, namespace, e.config.DefaultDomain)
	return &AppStatus{
		Name:        appName,
		Namespace:   namespace,
		Status:      inspect.State.Status,
		ContainerID: inspect.ID,
		Image:       inspect.Config.Image,
		URL:         fmt.Sprintf("http://%s", hostDomain),
		Restarts:    inspect.State.RestartCount,
	}, nil
}

// DeleteApp removes the app container
func (e *DefaultDockerEngine) DeleteApp(ctx context.Context, namespace, appName string) error {
	client := NewDockerRawClient(e.socketPath)
	containerName := e.ContainerName(namespace, appName, 1)
	return e.stopAndRemoveContainer(ctx, client, containerName)
}

// ScaleApp handles replica count
func (e *DefaultDockerEngine) ScaleApp(ctx context.Context, namespace, appName string, replicas int) error {
	// Replicas can be mapped to multiple containers or swarm/compose
	return nil
}

// StreamLogs streams container logs
func (e *DefaultDockerEngine) StreamLogs(ctx context.Context, namespace, appName string, follow bool, out io.Writer) error {
	client := NewDockerRawClient(e.socketPath)
	containerName := e.ContainerName(namespace, appName, 1)

	followStr := "false"
	if follow {
		followStr = "true"
	}
	logURL := fmt.Sprintf("/containers/%s/logs?stdout=true&stderr=true&follow=%s&tail=200", containerName, followStr)

	resp, err := client.Request(ctx, http.MethodGet, logURL, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// ListApps returns all active Dopinio apps in a namespace
func (e *DefaultDockerEngine) ListApps(ctx context.Context, namespace string) ([]AppStatus, error) {
	client := NewDockerRawClient(e.socketPath)
	url := "/containers/json?all=true"

	resp, err := client.Request(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var containers []struct {
		ID     string            `json:"Id"`
		Names  []string          `json:"Names"`
		Image  string            `json:"Image"`
		State  string            `json:"State"`
		Status string            `json:"Status"`
		Labels map[string]string `json:"Labels"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, err
	}

	var results []AppStatus
	for _, c := range containers {
		if c.Labels["dopinio.managed"] == "true" {
			appNs := c.Labels["dopinio.namespace"]
			appName := c.Labels["dopinio.app"]

			if namespace != "" && appNs != namespace {
				continue
			}

			domain := fmt.Sprintf("%s.%s.%s", appName, appNs, e.config.DefaultDomain)
			results = append(results, AppStatus{
				Name:        appName,
				Namespace:   appNs,
				Status:      c.State,
				ContainerID: c.ID,
				Image:       c.Image,
				URL:         fmt.Sprintf("http://%s", domain),
			})
		}
	}

	return results, nil
}

