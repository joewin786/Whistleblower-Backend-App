#!/bin/bash

# Whistleblower API - Kubernetes Deployment Script
# This script deploys the application to Kubernetes cluster

set -e

# Configuration
NAMESPACE="whistleblower"
K8S_DIR="k8s"

echo "🚀 Deploying Whistleblower API to Kubernetes"
echo "============================================="
echo ""

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed. Please install kubectl first."
    exit 1
fi

# Check if connected to cluster
if ! kubectl cluster-info &> /dev/null; then
    echo "❌ Not connected to a Kubernetes cluster. Please configure kubectl."
    exit 1
fi

echo "✅ Connected to Kubernetes cluster"
echo ""

# Create namespace if it doesn't exist
echo "📦 Creating namespace: ${NAMESPACE}"
kubectl apply -f ${K8S_DIR}/namespace.yaml

echo ""
echo "🔧 Applying ConfigMap..."
kubectl apply -f ${K8S_DIR}/configmap.yaml

echo ""
echo "🔐 Applying Secrets..."
kubectl apply -f ${K8S_DIR}/secret.yaml
kubectl apply -f ${K8S_DIR}/firebase-secret.yaml

echo ""
echo "💾 Deploying PostgreSQL..."
kubectl apply -f ${K8S_DIR}/postgres-pvc.yaml
kubectl apply -f ${K8S_DIR}/postgres-deployment.yaml
kubectl apply -f ${K8S_DIR}/postgres-service.yaml

echo ""
echo "⏳ Waiting for PostgreSQL to be ready..."
kubectl wait --for=condition=ready pod -l app=postgres -n ${NAMESPACE} --timeout=300s

echo ""
echo "🚀 Deploying Backend API..."
kubectl apply -f ${K8S_DIR}/backend-deployment.yaml
kubectl apply -f ${K8S_DIR}/backend-service.yaml

echo ""
echo "⏳ Waiting for Backend to be ready..."
kubectl wait --for=condition=ready pod -l app=wb-api -n ${NAMESPACE} --timeout=300s

echo ""
echo "✅ Deployment completed successfully!"
echo ""
echo "📊 Deployment Status:"
echo "===================="
kubectl get all -n ${NAMESPACE}

echo ""
echo "🔍 To check logs:"
echo "   kubectl logs -n ${NAMESPACE} -l app=wb-api -f"
echo ""
echo "🌐 To get service URL:"
echo "   kubectl get svc -n ${NAMESPACE} wb-api-service"
echo ""
