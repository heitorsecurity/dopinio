# Dopinio 🐳

> **Da sua máquina para a URL com um único comando — 100% no Docker, sem o peso do Kubernetes.**

O **Dopinio** é uma plataforma PaaS para desenvolvedores projetada para ser **extremanente fácil e intuitiva**.

Chega de sofrer com clusters Kubernetes, arquivos YAML quilométricos, Ingress Controllers quebrados ou Cert-Managers difíceis de configurar. No Dopinio, basta ter o **Docker** e rodar **um único comando**.

---

## ⚡ Instalação com Um Único Comando (CURL)

### No Linux / macOS / WSL:
```bash
curl -fsSL https://raw.githubusercontent.com/heitorsecurity/dopinio/main/install.sh | bash
```

### No Windows (PowerShell):
```powershell
irm https://raw.githubusercontent.com/heitorsecurity/dopinio/main/install.ps1 | iex
```

### 🎯 O que o instalador faz automaticamente por você:
1. **Verifica o Docker**: Checa se o Docker e o Docker Compose estão instalados e ativos.
2. **Cria a pasta**: Cria o diretório `~/.dopinio` limpo e isolado.
3. **Gera Credenciais Seguras**: Cria um arquivo `.env` com senhas seguras geradas na hora.
4. **Sobe toda a Infraestrutura**: Inicia a API do Dopinio, o Traefik (proxy dinâmico), Registry local e MinIO.
5. **Gera suas URLs**: Mostra na hora na tela todos os links e credenciais prontos para clicar e usar.

---

## 🖥️ Painéis e Acessos Padrão

Assim que o instalador termina, os seguintes serviços já estão no ar:

| Serviço | Endereço | Para que serve? |
| :--- | :--- | :--- |
| **API do Dopinio** | `http://api.dopinio.localhost:8443` | O cérebro do Dopinio que gerencia seus contêineres |
| **Traefik Proxy** | `http://localhost:80` | Encaminha o tráfego e gera domínios automaticamente |
| **Painel Traefik** | `http://localhost:8080` | Interface visual de rotas ativas e saúde dos serviços |
| **Registro Docker** | `http://localhost:5000` | Armazena as imagens criadas dos seus projetos |
| **MinIO Console** | `http://localhost:9001` | Guarda os arquivos de código e artefatos de build |

---

## 🚀 Como Publicar seu Projeto (Code to URL)

Você não precisa de Dockerfile nem de arquivos complexos.

1. Abra o terminal na pasta de qualquer projeto seu (Node, Python, Go, PHP, etc.):
   ```bash
   cd meu-projeto
   ```

2. Envie para o Dopinio:
   ```bash
   dopinio push --name meu-app
   ```

3. **Pronto!** O Dopinio constrói a imagem, sobe o contêiner e te entrega a URL:
   👉 `http://meu-app.workspace.dopinio.localhost`

---

## 🕹️ Comandos Úteis do Dia a Dia

Dentro de `~/.dopinio`:
```bash
# Ver todos os contêineres e status
docker compose ps

# Acompanhar os logs de tudo em tempo real
docker compose logs -f

# Parar a plataforma
docker compose down

# Iniciar novamente quando reiniciar a máquina
docker compose up -d
```

---

## 🗺️ Roteiro de Fases do Dopinio

```mermaid
flowchart LR
    Fase1["Fase 1: One-Line Curl & Fundação ✅"] --> Fase2["Fase 2: Motor Docker Nativo ⏳"]
    Fase2 --> Fase3["Fase 3: CLI 'Zero Dúvidas' 🛠️"]
    Fase3 --> Fase4["Fase 4: Dashboard Web Simples 🌐"]
```

### ✅ Fase 1: Fundação & Instalador de Linha Única (Concluída)
- Scripts de instalação automática via `curl` e PowerShell (`install.sh` e `install.ps1`).
- Verificação inteligente de requisitos e geração automática de senhas.
- Branches antigas desplugadas e backup histórico do Epinio original preservado em `epinio-original-full-backup.zip`.

### ⏳ Fase 2: Motor Docker Nativo (`internal/dockerengine`)
- Conexão nativa e direta com o Docker Engine via `/var/run/docker.sock`.
- Ciclo de vida completo dos apps (criar, iniciar, escalar, parar e deletar).
- Atribuição automática de portas e labels do Traefik.

### 🛠️ Fase 3: CLI "Zero Dúvidas"
- Comandos intuitivos: `dopinio up`, `dopinio push`, `dopinio apps`, `dopinio logs <app>`.

### 🌐 Fase 4: Dashboard Web Integrado
- Interface moderna no navegador para ver suas aplicações, consumo de memória/CPU e logs sem precisar abrir terminal.

---

## 💾 Projeto Forkado & Backup do Epinio Original

> **Nota de Arquivamento:** Este projeto é um fork do Epinio reimaginado para o ecossistema Docker. Um backup completo contendo todo o histórico e branches do Epinio original está compactado e disponível no arquivo:
> 📦 [`epinio-original-full-backup.zip`](./epinio-original-full-backup.zip)

## 🤝 Licença
Distribuído sob a licença Apache-2.0. Baseado no projeto Epinio.
