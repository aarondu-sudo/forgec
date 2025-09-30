Param()
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$OutDir = Join-Path $Root 'dist'
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$Lib = Join-Path $OutDir 'libmyapi.dll'
Write-Host "Building $Lib"
go build -buildmode=c-shared -o $Lib $Root
Write-Host "OK -> $Lib"
