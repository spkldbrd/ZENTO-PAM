$Root = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
Set-Location $Root
docker compose -f infra/docker/docker-compose.yml down
