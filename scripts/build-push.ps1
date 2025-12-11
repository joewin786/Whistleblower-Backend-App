# Whistleblower API - PowerShell Build and Push Script
# This script builds the Docker image and pushes it to your container registry

param(
    [string]$Tag = "latest",
    [string]$Registry = "your-registry"  # Update this: e.g., docker.io/username, gcr.io/project-id
)

$ImageName = "wb-api"
$FullImage = "$Registry/${ImageName}:$Tag"

Write-Host "🔨 Building Docker image: $FullImage" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# Build the Docker image
docker build -t $FullImage .

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Build failed!" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "✅ Build completed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "📦 Pushing image to registry: $FullImage" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan

# Push to registry
docker push $FullImage

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Push failed!" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "✅ Image pushed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "📝 Image details:" -ForegroundColor Yellow
Write-Host "   Name: $FullImage"
$imageSize = docker images $FullImage --format "{{.Size}}"
Write-Host "   Size: $imageSize"
Write-Host ""
Write-Host "🚀 To deploy this image to Kubernetes, update backend-deployment.yaml with:" -ForegroundColor Yellow
Write-Host "   image: $FullImage"
Write-Host ""
