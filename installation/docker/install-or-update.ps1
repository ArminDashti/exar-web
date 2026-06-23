#Requires -Version 5.1
$ErrorActionPreference = 'Stop'

$ImageName = if ($env:IMAGE_NAME) { $env:IMAGE_NAME } else { 'exar-web:latest' }
$ContainerName = if ($env:CONTAINER_NAME) { $env:CONTAINER_NAME } else { 'exar-web' }
$AppPort = if ($env:APP_PORT) { $env:APP_PORT } else { '8080' }
$DataDir = if ($env:DATA_DIR) { $env:DATA_DIR } else { Join-Path $env:LOCALAPPDATA 'exar-web\data' }

$ProjectRoot = Resolve-Path (Join-Path $PSScriptRoot '..\..')
$BinDir = Join-Path $ProjectRoot 'bin'
$ServerBinary = Join-Path $BinDir 'server'
$DistDir = Join-Path $ProjectRoot 'dist'

function Write-Log([string]$Message) {
    Write-Host "[docker-install] $Message"
}

function Write-Err([string]$Message) {
    Write-Error "[docker-install] $Message"
}

function Need-Command([string]$Name) {
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        Write-Err "missing required command: $Name"
        exit 1
    }
}

Need-Command docker
Need-Command go
Need-Command npm

try {
    docker info *> $null
    if ($LASTEXITCODE -ne 0) {
        Write-Err 'docker daemon is not running'
        exit 1
    }
}
catch {
    Write-Err 'docker is not accessible on this host'
    exit 1
}

New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
New-Item -ItemType Directory -Force -Path $DataDir | Out-Null

Write-Log 'Building frontend assets ...'
Push-Location $ProjectRoot
try {
    $viteBinary = Join-Path $ProjectRoot 'node_modules\vite\bin\vite.js'
    if (-not (Test-Path $viteBinary)) {
        npm install
        if ($LASTEXITCODE -ne 0) {
            Write-Err 'npm install failed'
            exit $LASTEXITCODE
        }
    }
    npm run build
    if ($LASTEXITCODE -ne 0) {
        Write-Err 'npm run build failed'
        exit $LASTEXITCODE
    }

    Write-Log 'Building backend binary ...'
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    go build -trimpath -ldflags '-s -w' -o $ServerBinary ./cmd/server
    if ($LASTEXITCODE -ne 0) {
        Write-Err 'go build failed'
        exit $LASTEXITCODE
    }
}
finally {
    Pop-Location
}

if (-not (Test-Path $ServerBinary)) {
    Write-Err "backend binary not found: $ServerBinary"
    exit 1
}
if (-not (Test-Path $DistDir)) {
    Write-Err "frontend build output not found: $DistDir"
    exit 1
}

Write-Log "Building Docker image $ImageName ..."
docker build -t $ImageName $ProjectRoot
if ($LASTEXITCODE -ne 0) {
    Write-Err 'docker build failed'
    exit $LASTEXITCODE
}

$existing = docker ps -a --format '{{.Names}}' | Where-Object { $_ -eq $ContainerName }
if ($existing) {
    Write-Log "Container $ContainerName exists. Replacing with updated image ..."
    docker rm -f $ContainerName | Out-Null
}
else {
    Write-Log "Container $ContainerName does not exist. Installing new container ..."
}

docker run -d `
    --name $ContainerName `
    --restart unless-stopped `
    -p "${AppPort}:8080" `
    -v "${DataDir}:/app/data" `
    $ImageName | Out-Null

if ($LASTEXITCODE -ne 0) {
    Write-Err 'docker run failed'
    exit $LASTEXITCODE
}

Write-Log "Done. Running container: $ContainerName on http://localhost:$AppPort"
exit 0
