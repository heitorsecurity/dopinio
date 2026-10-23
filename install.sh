#!/usr/bin/env bash
# ==============================================================================
#  🐳 Dopinio Installer - Da sua máquina para a URL com um único comando
#  Uso: curl -fsSL https://raw.githubusercontent.com/heitorsecurity/dopinio/main/install.sh | bash
# ==============================================================================

set -euo pipefail

# Cores e Estilos
BOLD='\033[1m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

clear 2>/dev/null || true

cat << "EOF"
  ____             _       _       
 |  _ \  ___  _ __ (_)_ __ (_) ___  
 | | | |/ _ \| '_ \| | '_ \| |/ _ \ 
 | |_| | (_) | |_) | | | | | | (_) |
 |____/ \___/| .__/|_|_| |_|_|\___/  🐳
             |_|                   
 Plataforma PaaS Simples, Nativa para Docker
EOF

echo -e "\n${CYAN}================================================================${NC}"
echo -e "${BOLD}Iniciando o instalador do Dopinio...${NC}"
echo -e "${CYAN}================================================================${NC}\n"

# ------------------------------------------------------------------------------
# 1. Verificações de Requisitos Básicos
# ------------------------------------------------------------------------------
echo -e "${BOLD}[1/5] Verificando requisitos...${NC}"

if ! command -v docker >/dev/null 2>&1; then
    echo -e "${RED}❌ Docker não foi encontrado na sua máquina!${NC}"
    echo -e "   Para rodar o Dopinio, por favor instale o Docker primeiro:"
    echo -e "   👉 https://docs.docker.com/get-docker/"
    exit 1
fi

if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}❌ O daemon do Docker não está rodando ou você não tem permissão!${NC}"
    echo -e "   Inicie o Docker Desktop ou serviço do Docker ('sudo systemctl start docker')."
    exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
    echo -e "${RED}❌ 'docker compose' não foi encontrado!${NC}"
    echo -e "   Certifique-se de que o plugin Compose do Docker está instalado."
    exit 1
fi

echo -e "${GREEN}✓ Docker e Docker Compose estão prontos e funcionando!${NC}\n"

# ------------------------------------------------------------------------------
# 2. Diretório de Instalação
# ------------------------------------------------------------------------------
INSTALL_DIR="${DOPINIO_DIR:-$HOME/.dopinio}"
echo -e "${BOLD}[2/5] Configurando ambiente em: ${CYAN}${INSTALL_DIR}${NC}..."

mkdir -p "${INSTALL_DIR}"
cd "${INSTALL_DIR}"

# ------------------------------------------------------------------------------
# 3. Baixando arquivos essenciais da stack
# ------------------------------------------------------------------------------
echo -e "${BOLD}[3/5] Baixando a stack leve do Dopinio...${NC}"

GITHUB_RAW="https://raw.githubusercontent.com/heitorsecurity/dopinio/main"

curl -fsSL "${GITHUB_RAW}/docker-compose.yml" -o docker-compose.yml
curl -fsSL "${GITHUB_RAW}/.env.example" -o .env.example

# Se não existir .env, cria um personalizado com senha segura aleatória
if [ ! -f .env ]; then
    RANDOM_PASS=$(tr -dc A-Za-z0-9 </dev/urandom | head -c 16 || echo "dopinio$(date +%s)")
    sed -e "s/change_this_secure_password/${RANDOM_PASS}/" .env.example > .env
    echo -e "${GREEN}✓ Arquivo .env configurado com credenciais seguras geradas automaticamente.${NC}"
else
    echo -e "${YELLOW}ℹ Arquivo .env existente preservado.${NC}"
fi

echo -e "${GREEN}✓ Arquivos de configuração prontos!${NC}\n"

# ------------------------------------------------------------------------------
# 4. Inicializando os Contêineres
# ------------------------------------------------------------------------------
echo -e "${BOLD}[4/5] Subindo a stack do Dopinio (Servidor, Traefik, Registry e MinIO)...${NC}"
echo -e "      ${CYAN}Isso pode levar alguns segundos na primeira vez...${NC}"

docker compose pull -q 2>/dev/null || true
docker compose up -d

echo -e "${GREEN}✓ Todos os contêineres iniciados com sucesso!${NC}\n"

# ------------------------------------------------------------------------------
# 5. Painel de Boas-vindas e URLs de Acesso
# ------------------------------------------------------------------------------
DOMAIN=$(grep '^DOPINIO_DOMAIN=' .env 2>/dev/null | cut -d '=' -f2 || echo "dopinio.localhost")
ADMIN_USER=$(grep '^DOPINIO_ADMIN_USER=' .env 2>/dev/null | cut -d '=' -f2 || echo "admin")
ADMIN_PASS=$(grep '^DOPINIO_ADMIN_PASSWORD=' .env 2>/dev/null | cut -d '=' -f2 || echo "change_this_secure_password")

echo -e "${CYAN}================================================================${NC}"
echo -e "${GREEN}${BOLD}🎉 Parabéns! O Dopinio foi instalado e já está rodando!${NC}"
echo -e "${CYAN}================================================================${NC}\n"

echo -e "${BOLD}🌐 SUAS URLs DE ACESSO:${NC}"
echo -e "   • ${BOLD}API do Dopinio:${NC}       ${CYAN}http://api.${DOMAIN}:8443${NC}"
echo -e "   • ${BOLD}Traefik Proxy:${NC}        ${CYAN}http://localhost:80${NC} (Roteia seus apps automaticamente)"
echo -e "   • ${BOLD}Painel Traefik:${NC}       ${CYAN}http://localhost:8080${NC}"
echo -e "   • ${BOLD}Registro de Imagens:${NC}  ${CYAN}http://localhost:5000${NC}"
echo -e "   • ${BOLD}MinIO Console:${NC}        ${CYAN}http://localhost:9001${NC}"
echo -e ""
echo -e "${BOLD}🔑 SUAS CREDENCIAIS:${NC}"
echo -e "   • Usuário: ${YELLOW}${ADMIN_USER}${NC}"
echo -e "   • Senha:   ${YELLOW}${ADMIN_PASS}${NC}"
echo -e ""
echo -e "${BOLD}🚀 COMO PUBLICAR SEU PRIMEIRO APP:${NC}"
echo -e "   Basta entrar na pasta de qualquer projeto seu e executar:"
echo -e "   ${CYAN}dopinio push --name meu-app${NC}"
echo -e "   E sua aplicação estará no ar em: ${GREEN}http://meu-app.workspace.${DOMAIN}${NC}\n"

echo -e "${BOLD}Comandos úteis:${NC}"
echo -e "   • Ver status:   ${CYAN}cd ${INSTALL_DIR} && docker compose ps${NC}"
echo -e "   • Ver logs:     ${CYAN}cd ${INSTALL_DIR} && docker compose logs -f${NC}"
echo -e "   • Parar tudo:   ${CYAN}cd ${INSTALL_DIR} && docker compose down${NC}"
echo -e "${CYAN}================================================================${NC}\n"
