#!/bin/bash

# Whistleblower API - Build and Push Docker Image Script
# This script builds the Docker image and pushes it to your container registry

set -e

# Configuration
IMAGE_NAME="wb-api"
REGISTRY="your-registry"  # Update this: e.g., docker.io/username, gcr.io/project-id, etc.
TAG="${1:-latest}"        # Use first argument as tag, default to 'latest'

FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${TAG}"

echo "🔨 Building Docker image: ${FULL_IMAGE}"
echo "=================================="

# Build the Docker image
docker build -t ${FULL_IMAGE} .

echo ""
echo "✅ Build completed successfully!"
echo ""
echo "📦 Pushing image to registry: ${FULL_IMAGE}"
echo "=================================="

# Push to registry
docker push ${FULL_IMAGE}

echo ""
echo "✅ Image pushed successfully!"
echo ""
echo "📝 Image details:"
echo "   Name: ${FULL_IMAGE}"
echo "   Size: $(docker images ${FULL_IMAGE} --format '{{.Size}}')"
echo ""
echo "🚀 To deploy this image to Kubernetes, update backend-deployment.yaml with:"
echo "   image: ${FULL_IMAGE}"
echo ""
