$Root = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
Set-Location $Root
if (Test-Path ".env") {
    docker compose -f infra/docker/docker-compose.yml --env-file .env up --build
} else {
    docker compose -f infra/docker/docker-compose.yml up --build
}
