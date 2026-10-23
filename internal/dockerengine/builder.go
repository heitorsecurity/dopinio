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
	"net/url"
)

// BuildOptions specifies parameters for building an application image
type BuildOptions struct {
	Namespace  string
	AppName    string
	Tag        string
	Dockerfile string
	TarContext io.Reader
}

// BuildImage uploads a tar context to Docker daemon and triggers image build
func (e *DefaultDockerEngine) BuildImage(ctx context.Context, opts BuildOptions, logWriter io.Writer) (string, error) {
	client := NewDockerRawClient(e.socketPath)

	imageTag := opts.Tag
	if imageTag == "" {
		imageTag = fmt.Sprintf("dopinio/%s-%s:latest", opts.Namespace, opts.AppName)
	}

	dockerfilePath := opts.Dockerfile
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

	buildURL := fmt.Sprintf("/build?t=%s&dockerfile=%s",
		url.QueryEscape(imageTag),
		url.QueryEscape(dockerfilePath),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s://%s/v1.45%s", client.hostScheme, client.hostAddr, buildURL),
		opts.TarContext,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-tar")

	resp, err := client.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("build request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("docker build failed (%d): %s", resp.StatusCode, string(respBody))
	}

	if logWriter != nil {
		_, _ = io.Copy(logWriter, resp.Body)
	}

	return imageTag, nil
}

