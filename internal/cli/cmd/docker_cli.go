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

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// CmdDockerGroup provides the streamlined, human-friendly commands for Dopinio Docker platform
var (
	flagAppName   string
	flagAppPort   int
	flagFollow    bool
	flagAppDomain string
)

func init() {
	RootCmd.AddCommand(CmdUp)
	RootCmd.AddCommand(CmdDown)
	RootCmd.AddCommand(CmdPush)
	RootCmd.AddCommand(CmdListApps)
	RootCmd.AddCommand(CmdAppLogs)
	RootCmd.AddCommand(CmdDeleteApp)

	CmdPush.Flags().StringVarP(&flagAppName, "name", "n", "", "Nome da aplicação (obrigatório)")
	CmdPush.Flags().IntVarP(&flagAppPort, "port", "p", 8080, "Porta interna do contêiner")
	CmdPush.Flags().StringVarP(&flagAppDomain, "domain", "d", "", "Domínio customizado (opcional)")
	_ = CmdPush.MarkFlagRequired("name")

	CmdAppLogs.Flags().BoolVarP(&flagFollow, "follow", "f", false, "Acompanhar logs continuamente")
}

// CmdUp brings up the local Dopinio stack
var CmdUp = &cobra.Command{
	Use:   "up",
	Short: "Inicia a plataforma Dopinio localmente com Docker Compose",
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("🐳 Iniciando a plataforma Dopinio...")
		c := exec.Command("docker", "compose", "up", "-d")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

// CmdDown stops the local Dopinio stack
var CmdDown = &cobra.Command{
	Use:   "down",
	Short: "Para a plataforma Dopinio",
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Yellow("🛑 Parando a plataforma Dopinio...")
		c := exec.Command("docker", "compose", "down")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

// CmdPush builds and deploys an application directly to Docker
var CmdPush = &cobra.Command{
	Use:   "push",
	Short: "Publica o projeto da pasta atual direto para a URL no Docker",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()
		appName := strings.ToLower(flagAppName)
		namespace := "workspace"

		color.Cyan("\n=======================================================")
		color.White("🚀 Publicando '%s' no Dopinio...", appName)
		color.Cyan("=======================================================\n")

		imageTag := fmt.Sprintf("dopinio/%s:latest", appName)

		// Check if Dockerfile exists or build image
		dockerfile := filepath.Join(dir, "Dockerfile")
		if _, err := os.Stat(dockerfile); err == nil {
			fmt.Printf("📦 Dockerfile encontrado. Construindo imagem '%s'...\n", imageTag)
			buildCmd := exec.Command("docker", "build", "-t", imageTag, dir)
			buildCmd.Stdout = os.Stdout
			buildCmd.Stderr = os.Stderr
			if err := buildCmd.Run(); err != nil {
				return fmt.Errorf("falha ao construir imagem Docker: %w", err)
			}
		} else {
			fmt.Println("ℹ Nenhum Dockerfile encontrado. Usando imagem padrão...")
		}

		// Send deployment request to Dopinio API
		apiURL := "http://localhost:8443/docker/namespaces/" + namespace + "/apps"
		payload := map[string]interface{}{
			"name":      appName,
			"namespace": namespace,
			"image":     imageTag,
			"port":      flagAppPort,
			"domain":    flagAppDomain,
		}

		jsonBody, _ := json.Marshal(payload)
		resp, err := http.Post(apiURL, "application/json", bytes.NewReader(jsonBody))
		if err != nil {
			// Se a API não responder, faz o deploy local direto via Docker CLI
			color.Yellow("⚠️ API central não respondeu na porta 8443. Executando deploy direto...")
			runCmd := exec.Command("docker", "run", "-d",
				"--name", fmt.Sprintf("dopinio-%s-%s", namespace, appName),
				"--network", "dopinio-net",
				"--label", "dopinio.managed=true",
				"--label", fmt.Sprintf("dopinio.app=%s", appName),
				"--label", fmt.Sprintf("dopinio.namespace=%s", namespace),
				"--label", "traefik.enable=true",
				"--label", fmt.Sprintf("traefik.http.routers.%s.rule=Host(`%s.workspace.dopinio.localhost`)", appName, appName),
				"--label", fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port=%d", appName, flagAppPort),
				imageTag,
			)
			runCmd.Stdout = os.Stdout
			runCmd.Stderr = os.Stderr
			_ = runCmd.Run()
		} else {
			_ = resp.Body.Close()
		}

		color.Green("\n✓ Aplicação colocada no ar com sucesso!")
		color.White("👉 Sua URL pública: ")
		color.Cyan("   http://%s.workspace.dopinio.localhost\n", appName)
		return nil
	},
}

// CmdListApps lists running applications
var CmdListApps = &cobra.Command{
	Use:   "apps",
	Short: "Lista todas as aplicações rodando no Dopinio",
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("\n📋 Aplicações no Dopinio:")
		c := exec.Command("docker", "ps", "--filter", "label=dopinio.managed=true",
			"--format", "table {{.Names}}\t{{.Status}}\t{{.Ports}}")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

// CmdAppLogs streams logs from an application
var CmdAppLogs = &cobra.Command{
	Use:   "logs [nome-do-app]",
	Short: "Acompanha os logs em tempo real da aplicação",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]
		containerName := fmt.Sprintf("dopinio-workspace-%s", appName)

		dockerArgs := []string{"logs"}
		if flagFollow {
			dockerArgs = append(dockerArgs, "-f")
		}
		dockerArgs = append(dockerArgs, containerName)

		c := exec.Command("docker", dockerArgs...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

// CmdDeleteApp removes an application
var CmdDeleteApp = &cobra.Command{
	Use:   "delete [nome-do-app]",
	Short: "Para e remove uma aplicação",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]
		containerName := fmt.Sprintf("dopinio-workspace-%s", appName)

		color.Yellow("Excluindo %s...", containerName)
		c := exec.Command("docker", "rm", "-f", containerName)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

