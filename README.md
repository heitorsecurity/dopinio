# Dopinio 🐳

> **From Code to URL directly on Docker** — lightweight, zero-k8s developer platform (PaaS).

Dopinio is a reimagined, Docker-native fork of Epinio. While upstream Epinio requires a full Kubernetes cluster (ingress controllers, cert-manager, Tekton, and Helm), **Dopinio is built to run directly on standard Docker and Docker Compose**.

---

## ⚡ Quick Start with Docker

### 1. Inicie a stack completa do Dopinio

```bash
# Clone e entre no diretório
cd dopinio

# Copie o arquivo de variáveis de ambiente
cp .env.example .env

# Suba os serviços (Dopinio API, Traefik, Registry e MinIO)
docker compose up -d
```

### 2. Acessos padrão

- **API Server & Dashboard:** `http://api.dopinio.localhost:8443`
- **Traefik Reverse Proxy:** `http://localhost:80` (HTTP) / `http://localhost:443` (HTTPS)
- **Traefik Dashboard:** `http://localhost:8080`
- **Registry Local:** `http://localhost:5000`
- **MinIO Console:** `http://localhost:9001`

---

## 🚀 Como Funciona o Dopinio no Docker

1. **Docker Engine Runtime (`internal/dockerengine`)**:
   - Conecta-se diretamente ao Docker daemon (`/var/run/docker.sock`).
   - Gerencia contêineres de aplicações, volumes e redes dedicadas (`dopinio-net`).
2. **Roteamento Dinâmico com Traefik**:
   - Ao criar uma aplicação via CLI (`dopinio push`), contêineres recebem labels do Traefik automaticamente.
   - Rotas sob demanda: `http://<app>.<namespace>.dopinio.localhost`.
3. **Builds de Código**:
   - Suporte a builds com `Dockerfile` ou Cloud Native Buildpacks direto no daemon do Docker.
4. **Armazenamento de Objetos & Registro**:
   - MinIO local para upload de tarballs de código-fonte.
   - Registry local (v2) para armazenamento de imagens geradas.

---
Once this is in place, and leaving out DNS setup, an installation reduces to:

```
helm repo add epinio https://epinio.github.io/helm-charts
helm repo update

helm install --namespace epinio --create-namespace epinio epinio/epinio \
  --set global.domain=mydomain.example.com
```

### CLI installation

Installation of the Epinio CLI is by downloading a binary from the
[release page](https://github.com/epinio/epinio/releases), or usage of `brew`, i.e.

```
brew install epinio
```

There is further documentation [here](https://docs.epinio.io/getting-started/install-cli).

### Quick Start Tutorial

Our [QuickStart Tutorial](https://docs.epinio.io/getting-started/quickstart) works through how to create a namespace and push an application.

### Reach Us

- Slack: #epinio on [Rancher Users](https://rancher-users.slack.com/)
- Github: [Discuss](https://github.com/epinio/epinio/discussions/new)

### Contributing

`Epinio` uses [Github Project](https://github.com/epinio/epinio/projects/1) for tracking issues.

Find more information in the [Contribution Guide](https://docs.epinio.io/contribute/contributing).

The [developer documentation](./docs) explains how to build and run Epinio from a git source checkout.

## Features

- **Security**
  - TLS secured API server
  - Basic Authentication to access the API
  - __or__ OIDC-based token
- **Epinio Clients**
  - Web UI
  - Epinio CLI
- **Apps**
  - Push code directly without further tools or steps
  - Basic operation of your application once pushed
  - Cloud Native Buildpacks build and containerize your code for you
- **Configurations**
  - CRUD operations on your configuration. A configuration can be a database, SaaS etc. A configuration can be an external component or can be created using `epinio configuration`.
  - Bind configurations to apps.

## Example apps

- Rails: https://github.com/epinio/example-rails
- Java: https://github.com/spring-projects/spring-petclinic/
- Paketo Buildpack example apps: https://github.com/paketo-buildpacks/samples

## License

Copyright (c) 2020-2024 [SUSE, LLC](https://suse.com)

Copyright (c) 2025-2026 [Krumware LLC](https://www.krum.io)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Breaking Changes & Migrations

### 1.12 to 1.13

Epinio 1.13 rehomes configurations for the staging workloads to a more standardized format that supports a larger variety of configs.  These are no longer configured directly on the Epinio container via ENV variables or through Viper CLI flags but rather read from an in-cluster ConfigMap at staging time.

To summarize, version 1.12 referenced the following ENV variables:

```bash
STAGING_SERVICE_ACCOUNT_NAME
STAGING_RESOURCE_CPU
STAGING_RESOURCE_MEMORY
STAGING_RESOURCE_DISK
```

These ENV variables were configured directly by the Helm Chart in the deployment's `env` section.  This is no longer the case as we push in the direction of improving configurability of these staging workloads.  These configurations are now more expansive, including resource requests vs. limits, node tolerations, affinity criteria, etc.  And as such, the ENV approach to this would become unruly as capabilities grew.  Instead, we are conforming more closely to how Kubernetes would structure these configs directly in the Job YAML.  These configurations are now read in from chart values into the stage script ConfigMaps.  The contents of this ConfigMap, as far as the Epinio Server is concerned, would resemble the following (with examples):

```yaml
serviceAccountName: "epinio-server"
ttlSecondsAfterFinished: 300
resources:
  requests:
    cpu: "200m"
    memory: "1Gi"
  # limits:
  #   cpu: ""
  #   memory: ""
storage:
  cache:
    emptyDir: false
    size: 1Gi
    storageClassName: ""
    volumeMode: Filesystem
    accessModes:
    - ReadWriteOnce
  sourceBlobs:
    emptyDir: true
    # size: 1Gi
    # storageClassName: ""
    # volumeMode: Filesystem
    # accessModes:
    # - ReadWriteOnce
nodeSelector: {}
  # kubernetes.io/os: linux
affinity: {}
  # nodeAffinity:
  #   requiredDuringSchedulingIgnoredDuringExecution:
  #     nodeSelectorTerms:
  #     - matchExpressions:
  #       - key: kubernetes.io/os
  #         operator: In
  #         values:
  #         - linux
tolerations: []
  # - key: "kubernetes.io/os"
  #   operator: "Equal"
  #   value: "linux"
  #   effect: "NoSchedule"
```

