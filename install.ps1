# ==============================================================================
#  🐳 Dopinio Installer para Windows (PowerShell)
#  Uso: irm https://raw.githubusercontent.com/heitorsecurity/dopinio/main/install.ps1 | iex
# ==============================================================================

$ErrorActionPreference = 'Stop'

Clear-Host
Write-Host @"
  ____             _       _       
 |  _ \  ___  _ __ (_)_ __ (_) ___  
 | | | |/ _ \| '_ \| | '_ \| |/ _ \ 
 | |_| | (_) | |_) | | | | | | (_) |
 |____/ \___/| .__/|_|_| |_|_|\___/  🐳
             |_|                   
 Plataforma PaaS Simples, Nativa para Docker
"@ -ForegroundColor Cyan

Write-Host "`n================================================================" -ForegroundColor Cyan
Write-Host "Iniciando o instalador do Dopinio para Windows..." -ForegroundColor Yellow
Write-Host "================================================================`n" -ForegroundColor Cyan

# 1. Checagem do Docker
Write-Host "[1/5] Verificando Docker Desktop..." -ForegroundColor White
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Host "❌ O Docker não foi encontrado na máquina!" -ForegroundColor Red
    Write-Host "   Instale o Docker Desktop: https://docs.docker.com/desktop/setup/install/windows-install/"
    exit 1
}

try {
    $null = docker info
} catch {
    Write-Host "❌ O Docker Desktop não está rodando!" -ForegroundColor Red
    Write-Host "   Por favor, abra o Docker Desktop e aguarde ele iniciar."
    exit 1
}
Write-Host "✓ Docker Desktop está ativo e pronto!`n" -ForegroundColor Green

# 2. Pasta de Instalação
$InstallDir = Join-Path $HOME ".dopinio"
Write-Host "[2/5] Configurando ambiente em: $InstallDir..." -ForegroundColor White
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}
Set-Location $InstallDir

# 3. Baixando arquivos essenciais
Write-Host "[3/5] Baixando a stack leve do Dopinio..." -ForegroundColor White
$BaseUri = "https://raw.githubusercontent.com/heitorsecurity/dopinio/main"

Invoke-WebRequest -Uri "$BaseUri/docker-compose.yml" -OutFile "docker-compose.yml" -UseBasicParsing
Invoke-WebRequest -Uri "$BaseUri/.env.example" -OutFile ".env.example" -UseBasicParsing

if (-not (Test-Path ".env")) {
    $RandomPass = -join ((65..90) + (97..122) + (48..57) | Get-Random -Count 16 | ForEach-Object {[char]$_})
    $EnvContent = Get-Content ".env.example" -Raw
    $EnvContent = $EnvContent -replace "change_this_secure_password", $RandomPass
    Set-Content -Path ".env" -Value $EnvContent
    Write-Host "✓ Arquivo .env criado com credenciais automáticas seguras." -ForegroundColor Green
} else {
    Write-Host "ℹ Arquivo .env existente preservado." -ForegroundColor Yellow
}
Write-Host "✓ Configurações prontas!`n" -ForegroundColor Green

# 4. Subindo contêineres
Write-Host "[4/5] Subindo a stack do Dopinio..." -ForegroundColor White
docker compose up -d

Write-Host "✓ Todos os contêineres iniciados com sucesso!`n" -ForegroundColor Green

# 5. Painel Final
$Domain = "dopinio.localhost"
$AdminUser = "admin"
$AdminPass = "change_this_secure_password"

if (Test-Path ".env") {
    Get-Content ".env" | ForEach-Object {
        if ($_ -match "^DOPINIO_DOMAIN=(.*)$") { $Domain = $matches[1] }
        if ($_ -match "^DOPINIO_ADMIN_USER=(.*)$") { $AdminUser = $matches[1] }
        if ($_ -match "^DOPINIO_ADMIN_PASSWORD=(.*)$") { $AdminPass = $matches[1] }
    }
}

Write-Host "================================================================" -ForegroundColor Cyan
Write-Host "🎉 Parabéns! O Dopinio foi instalado e já está rodando!" -ForegroundColor Green
Write-Host "================================================================`n" -ForegroundColor Cyan

Write-Host "🌐 SUAS URLs DE ACESSO:" -ForegroundColor Yellow
Write-Host "   • API do Dopinio:       http://api.$Domain`:8443" -ForegroundColor White
Write-Host "   • Traefik Proxy:        http://localhost:80 (Roteia seus apps)" -ForegroundColor White
Write-Host "   • Painel Traefik:       http://localhost:8080" -ForegroundColor White
Write-Host "   • Registro de Imagens:  http://localhost:5000" -ForegroundColor White
Write-Host "   • MinIO Console:        http://localhost:9001`n" -ForegroundColor White

Write-Host "🔑 SUAS CREDENCIAIS:" -ForegroundColor Yellow
Write-Host "   • Usuário: $AdminUser" -ForegroundColor White
Write-Host "   • Senha:   $AdminPass`n" -ForegroundColor White

Write-Host "🚀 COMO PUBLICAR SEU PRIMEIRO APP:" -ForegroundColor Yellow
Write-Host "   Abra qualquer pasta de código e execute:"
Write-Host "   dopinio push --name meu-app" -ForegroundColor Cyan
Write-Host "   Sua URL estará no ar em: http://meu-app.workspace.$Domain`n" -ForegroundColor Green

Write-Host "Comandos úteis:" -ForegroundColor DarkGray
Write-Host "   • Ver status:   cd $InstallDir; docker compose ps" -ForegroundColor DarkGray
Write-Host "   • Ver logs:     cd $InstallDir; docker compose logs -f" -ForegroundColor DarkGray
Write-Host "   • Parar tudo:   cd $InstallDir; docker compose down" -ForegroundColor DarkGray
Write-Host "================================================================`n" -ForegroundColor Cyan
