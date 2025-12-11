# Kubernetes Deployment Guide - Whistleblower API

Panduan lengkap untuk deploy Whistleblower REST API ke Kubernetes cluster dengan PostgreSQL database.

## 📋 Prerequisites

- Kubernetes cluster (minikube, GKE, EKS, AKS, atau lainnya)
- `kubectl` installed dan configured
- Docker installed (untuk build image)
- Container registry access (Docker Hub, GCR, ECR, dll)

## 🏗️ Architecture

```
┌─────────────────────────────────────────┐
│         Kubernetes Cluster              │
│                                         │
│  ┌──────────────┐    ┌──────────────┐  │
│  │   wb-api     │───▶│  PostgreSQL  │  │
│  │  (2 replicas)│    │ (StatefulSet)│  │
│  └──────────────┘    └──────────────┘  │
│         │                               │
│         ▼                               │
│  ┌──────────────┐                      │
│  │ LoadBalancer │                      │
│  │   Service    │                      │
│  └──────────────┘                      │
└─────────────────────────────────────────┘
```

## 📁 File Structure

```
k8s/
├── namespace.yaml              # Namespace definition
├── configmap.yaml              # Non-sensitive config
├── secret.yaml                 # Sensitive credentials
├── firebase-secret.yaml        # Firebase service account
├── postgres-pvc.yaml           # PostgreSQL storage
├── postgres-deployment.yaml    # PostgreSQL StatefulSet
├── postgres-service.yaml       # PostgreSQL internal service
├── backend-deployment.yaml     # Backend API deployment
├── backend-service.yaml        # Backend LoadBalancer service
├── ingress.yaml               # (Optional) Ingress for domain routing
└── kustomization.yaml         # Kustomize config
```

## 🚀 Quick Start

### 1. Build dan Push Docker Image

```bash
# Update REGISTRY di scripts/build-push.sh dengan registry Anda
# Contoh: docker.io/username, gcr.io/project-id, etc.

# Build dan push image
chmod +x scripts/build-push.sh
./scripts/build-push.sh latest
```

### 2. Update Backend Deployment

Edit `k8s/backend-deployment.yaml` dan update image:

```yaml
spec:
  template:
    spec:
      containers:
      - name: wb-api
        image: your-registry/wb-api:latest  # Update ini
```

### 3. Deploy ke Kubernetes

```bash
# Gunakan deployment script
chmod +x scripts/deploy.sh
./scripts/deploy.sh
```

Atau manual:

```bash
# Apply semua manifests
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml
kubectl apply -f k8s/firebase-secret.yaml
kubectl apply -f k8s/postgres-pvc.yaml
kubectl apply -f k8s/postgres-deployment.yaml
kubectl apply -f k8s/postgres-service.yaml
kubectl apply -f k8s/backend-deployment.yaml
kubectl apply -f k8s/backend-service.yaml
```

### 4. Verify Deployment

```bash
# Check all resources
kubectl get all -n whistleblower

# Check pods status
kubectl get pods -n whistleblower

# Check logs
kubectl logs -n whistleblower -l app=wb-api -f

# Check service
kubectl get svc -n whistleblower wb-api-service
```

## 🔧 Configuration

### Environment Variables

**ConfigMap** (`configmap.yaml`) - Non-sensitive:
- `SMTP_HOST`: SMTP server
- `DB_DRIVER`: Database driver (postgres)
- `FIREBASE_SERVICE_ACCOUNT_PATH`: Path to Firebase config
- `PORT`: Application port

**Secret** (`secret.yaml`) - Sensitive (base64 encoded):
- `GOOGLE_CLIENT_ID`: Google OAuth client ID
- `JWT_SECRET`: JWT signing secret
- `SMTP_USER`: SMTP username
- `SMTP_PASS`: SMTP password
- `GEMINI_API_KEY`: Gemini API key
- `GEMINI_API_KEY_CHAT`: Gemini chat API key
- `DB_SOURCE`: PostgreSQL connection string

### Database Configuration

PostgreSQL runs as a StatefulSet with:
- **Image**: `postgres:16-alpine`
- **Storage**: 10Gi PersistentVolume
- **Database**: `Whistleblower_db`
- **User**: `postgres`
- **Password**: `123456` (⚠️ Change in production!)

Connection string dalam cluster:
```
postgres://postgres:123456@postgres-service:5432/Whistleblower_db?sslmode=disable
```

## 🔐 Security Notes

> [!WARNING]
> File `secret.yaml` dan `firebase-secret.yaml` berisi kredensial sensitif!
> - **JANGAN** commit ke Git repository
> - Gunakan `.gitignore` untuk exclude file ini
> - Untuk production, gunakan secret management tools seperti:
>   - Sealed Secrets
>   - External Secrets Operator
>   - HashiCorp Vault
>   - Cloud provider secret managers (AWS Secrets Manager, GCP Secret Manager, etc.)

## 🌐 Accessing the API

### LoadBalancer Service

Jika menggunakan LoadBalancer:

```bash
# Get external IP
kubectl get svc -n whistleblower wb-api-service

# API akan accessible di:
# http://<EXTERNAL-IP>:80
```

### NodePort (Alternative)

Jika cluster tidak support LoadBalancer, edit `backend-service.yaml`:

```yaml
spec:
  type: NodePort
  ports:
  - port: 8080
    targetPort: 8080
    nodePort: 30080
```

Access via: `http://<NODE-IP>:30080`

### Ingress (Recommended for Production)

Untuk production dengan domain:

1. Install Ingress Controller (nginx):
```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.1/deploy/static/provider/cloud/deploy.yaml
```

2. Update `ingress.yaml` dengan domain Anda:
```yaml
spec:
  rules:
  - host: api.yourdomain.com  # Update ini
```

3. Apply ingress:
```bash
kubectl apply -f k8s/ingress.yaml
```

## 📊 Monitoring & Debugging

### View Logs

```bash
# Backend logs
kubectl logs -n whistleblower -l app=wb-api -f

# PostgreSQL logs
kubectl logs -n whistleblower -l app=postgres -f

# Specific pod
kubectl logs -n whistleblower <pod-name> -f
```

### Execute Commands in Pod

```bash
# Access backend pod
kubectl exec -it -n whistleblower <wb-api-pod-name> -- sh

# Access PostgreSQL
kubectl exec -it -n whistleblower <postgres-pod-name> -- psql -U postgres -d Whistleblower_db
```

### Port Forward (Local Testing)

```bash
# Forward backend port
kubectl port-forward -n whistleblower svc/wb-api-service 8080:80

# Access at http://localhost:8080
```

### Check Resource Usage

```bash
# Pod resource usage
kubectl top pods -n whistleblower

# Node resource usage
kubectl top nodes
```

## 🔄 Updates & Rollbacks

### Update Application

```bash
# Build new image with version tag
./scripts/build-push.sh v1.0.1

# Update deployment
kubectl set image deployment/wb-api wb-api=your-registry/wb-api:v1.0.1 -n whistleblower

# Check rollout status
kubectl rollout status deployment/wb-api -n whistleblower
```

### Rollback

```bash
# Rollback to previous version
kubectl rollout undo deployment/wb-api -n whistleblower

# Rollback to specific revision
kubectl rollout undo deployment/wb-api --to-revision=2 -n whistleblower

# Check rollout history
kubectl rollout history deployment/wb-api -n whistleblower
```

## 🧹 Cleanup

### Delete All Resources

```bash
# Delete namespace (will delete all resources inside)
kubectl delete namespace whistleblower
```

### Delete Specific Resources

```bash
# Delete backend only
kubectl delete -f k8s/backend-deployment.yaml
kubectl delete -f k8s/backend-service.yaml

# Delete database (⚠️ Data will be lost!)
kubectl delete -f k8s/postgres-deployment.yaml
kubectl delete -f k8s/postgres-service.yaml
kubectl delete -f k8s/postgres-pvc.yaml
```

## 🔧 Troubleshooting

### Pods Not Starting

```bash
# Describe pod for events
kubectl describe pod -n whistleblower <pod-name>

# Check events
kubectl get events -n whistleblower --sort-by='.lastTimestamp'
```

### Database Connection Issues

```bash
# Check if PostgreSQL is ready
kubectl get pods -n whistleblower -l app=postgres

# Test connection from backend pod
kubectl exec -it -n whistleblower <wb-api-pod> -- nc -zv postgres-service 5432
```

### Image Pull Errors

```bash
# Check if image exists
docker pull your-registry/wb-api:latest

# Create image pull secret if using private registry
kubectl create secret docker-registry regcred \
  --docker-server=<your-registry> \
  --docker-username=<username> \
  --docker-password=<password> \
  -n whistleblower

# Add to deployment
spec:
  imagePullSecrets:
  - name: regcred
```

## 📚 Additional Resources

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)
- [PostgreSQL on Kubernetes](https://www.postgresql.org/docs/)
- [Ingress NGINX](https://kubernetes.github.io/ingress-nginx/)

## 🆘 Support

Untuk pertanyaan atau issues, silakan buat issue di repository atau hubungi tim development.
