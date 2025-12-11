# Whistleblower API - PowerShell Deployment Script
# This script deploys the application to Kubernetes cluster

$Namespace = "whistleblower"
$K8sDir = "k8s"

Write-Host "🚀 Deploying Whistleblower API to Kubernetes" -ForegroundColor Cyan
Write-Host "=============================================" -ForegroundColor Cyan
Write-Host ""

# Check if kubectl is installed
if (-not (Get-Command kubectl -ErrorAction SilentlyContinue)) {
    Write-Host "❌ kubectl is not installed. Please install kubectl first." -ForegroundColor Red
    exit 1
}

# Check if connected to cluster
kubectl cluster-info 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Not connected to a Kubernetes cluster. Please configure kubectl." -ForegroundColor Red
    exit 1
}

Write-Host "✅ Connected to Kubernetes cluster" -ForegroundColor Green
Write-Host ""

# Create namespace if it doesn't exist
Write-Host "📦 Creating namespace: $Namespace" -ForegroundColor Cyan
kubectl apply -f "$K8sDir/namespace.yaml"

Write-Host ""
Write-Host "🔧 Applying ConfigMap..." -ForegroundColor Cyan
kubectl apply -f "$K8sDir/configmap.yaml"

Write-Host ""
Write-Host "🔐 Applying Secrets..." -ForegroundColor Cyan
kubectl apply -f "$K8sDir/secret.yaml"
kubectl apply -f "$K8sDir/firebase-secret.yaml"

Write-Host ""
Write-Host "💾 Deploying PostgreSQL..." -ForegroundColor Cyan
kubectl apply -f "$K8sDir/postgres-pvc.yaml"
kubectl apply -f "$K8sDir/postgres-deployment.yaml"
kubectl apply -f "$K8sDir/postgres-service.yaml"

Write-Host ""
Write-Host "⏳ Waiting for PostgreSQL to be ready..." -ForegroundColor Yellow
kubectl wait --for=condition=ready pod -l app=postgres -n $Namespace --timeout=300s

Write-Host ""
Write-Host "🚀 Deploying Backend API..." -ForegroundColor Cyan
kubectl apply -f "$K8sDir/backend-deployment.yaml"
kubectl apply -f "$K8sDir/backend-service.yaml"

Write-Host ""
Write-Host "⏳ Waiting for Backend to be ready..." -ForegroundColor Yellow
kubectl wait --for=condition=ready pod -l app=wb-api -n $Namespace --timeout=300s

Write-Host ""
Write-Host "✅ Deployment completed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "📊 Deployment Status:" -ForegroundColor Yellow
Write-Host "====================" -ForegroundColor Yellow
kubectl get all -n $Namespace

Write-Host ""
Write-Host "🔍 To check logs:" -ForegroundColor Yellow
Write-Host "   kubectl logs -n $Namespace -l app=wb-api -f"
Write-Host ""
Write-Host "🌐 To get service URL:" -ForegroundColor Yellow
Write-Host "   kubectl get svc -n $Namespace wb-api-service"
Write-Host ""
