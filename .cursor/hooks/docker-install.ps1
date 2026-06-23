#Requires -Version 5.1
$ErrorActionPreference = 'Stop'

$stdin = [Console]::In.ReadToEnd()
if ([string]::IsNullOrWhiteSpace($stdin)) {
    exit 0
}

try {
    $payload = $stdin | ConvertFrom-Json
}
catch {
    exit 0
}

if ($payload.status -and $payload.status -ne 'completed') {
    exit 0
}

$installScript = Join-Path $PSScriptRoot '..\..\installation\docker\install-or-update.ps1'
$installScript = (Resolve-Path $installScript).Path

& $installScript
exit $LASTEXITCODE
