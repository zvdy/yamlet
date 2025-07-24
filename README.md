# Yamlet 🎯

[![Go Report Card](https://goreportcard.com/badge/github.com/zvdy/yamlet)](https://goreportcard.com/report/github.com/zvdy/yamlet)
[![Docker Hub](https://img.shields.io/docker/v/zvdy/yamlet?label=docker&logo=docker)](https://hub.docker.com/r/zvdy/yamlet)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

<div align="center">
   <img src="https://i.imgur.com/L7Pn0Pn.png" alt="yamlet" width="260">
   <p><em>Lightweight, distributed key-value store for YAML configurations in Kubernetes</em></p>
</div>

A production-ready, cloud-native configuration management service designed for Kubernetes environments. Yamlet provides secure, namespace-isolated YAML configuration storage with token-based authentication and runtime token management.

## ✨ Features

- 🏗️ **Namespace Isolation**: Per-namespace configuration storage with token-based access control
- 🔐 **Dynamic Token Management**: Runtime token creation, revocation, and management via admin API
- 💾 **Flexible Storage**: In-memory or persistent file-based storage backends
- 🚀 **Kubernetes Native**: Optimized for cloud-native deployments with health checks and probes
- 📡 **RESTful API**: Clean HTTP API for configuration management
- 🔧 **Admin Interface**: Administrative endpoints for token lifecycle management
- 🏥 **Production Ready**: Comprehensive logging, error handling, and monitoring support
- 🐳 **Docker Ready**: Available on Docker Hub with versioned releases

## 🚀 Quick Start

### Option 1: Docker Hub (Recommended)

```bash
# Pull and run from Docker Hub
docker run -p 8080:8080 zvdy/yamlet:0.0.1

# Test the API
curl -H "Authorization: Bearer dev-token" http://localhost:8080/health
```

### Option 2: Kubernetes Deployment

```bash
# Deploy to Kubernetes with dedicated namespace
kubectl apply -f https://raw.githubusercontent.com/zvdy/yamlet/main/k8s/yamlet-namespace.yaml

# Or deploy to Minikube
kubectl apply -f https://raw.githubusercontent.com/zvdy/yamlet/main/k8s/minikube.yaml
```

### Option 3: Local Development

```bash
git clone https://github.com/zvdy/yamlet.git
cd yamlet
go build -o yamlet ./cmd/yamlet
./yamlet
```

## 📋 API Reference

### 🔑 Authentication

All API requests require a bearer token:
```
Authorization: Bearer <token>
```

### 🎯 Core Endpoints

#### Configuration Management
```bash
# Store configuration
POST /namespaces/{namespace}/configs/{name}
curl -X POST -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/x-yaml" \
  --data "app: myapp\nversion: 1.0" \
  http://localhost:8080/namespaces/dev/configs/app.yaml

# Retrieve configuration
GET /namespaces/{namespace}/configs/{name}
curl -H "Authorization: Bearer dev-token" \
  http://localhost:8080/namespaces/dev/configs/app.yaml

# List configurations
GET /namespaces/{namespace}/configs
curl -H "Authorization: Bearer dev-token" \
  http://localhost:8080/namespaces/dev/configs

# Delete configuration
DELETE /namespaces/{namespace}/configs/{name}
curl -X DELETE -H "Authorization: Bearer dev-token" \
  http://localhost:8080/namespaces/dev/configs/app.yaml
```

#### Admin Token Management
```bash
# List all tokens (admin only)
GET /admin/tokens
curl -H "Authorization: Bearer admin-secret-token-change-me" \
  http://localhost:8080/admin/tokens

# Create new token (admin only)
POST /admin/tokens
curl -X POST -H "Authorization: Bearer admin-secret-token-change-me" \
  -H "Content-Type: application/json" \
  --data '{"token": "new-token", "namespace": "production"}' \
  http://localhost:8080/admin/tokens

# Revoke token (admin only)
DELETE /admin/tokens/{token}
curl -X DELETE -H "Authorization: Bearer admin-secret-token-change-me" \
  http://localhost:8080/admin/tokens/old-token
```

#### Health & Monitoring
```bash
# Health check
GET /health
curl http://localhost:8080/health
```

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `USE_FILES` | `false` | Enable persistent file storage |
| `DATA_DIR` | `/data` | Storage directory for file backend |
| `YAMLET_ADMIN_TOKEN` | `admin-secret-token-change-me` | Admin token for management operations |
| `YAMLET_TOKENS` | `dev-token:dev,test-token:test` | Initial token:namespace mappings |

### Default Tokens

| Token | Namespace | Purpose |
|-------|-----------|---------|
| `dev-token` | `dev` | Development environment |
| `test-token` | `test` | Testing environment |

**⚠️ Important**: Change the admin token in production via `YAMLET_ADMIN_TOKEN` environment variable.

## 🏗️ Architecture & Examples

### Example Applications

The repository includes complete example applications demonstrating Yamlet integration:

```
examples/
├── mock-db/           # Go-based mock database service
├── sample-app/        # Application that fetches config from Yamlet
└── README.md          # Detailed example documentation
```

### Configuration-as-Code Pattern

```yaml
# Example: Application configuration stored in Yamlet
app: my-microservice
version: 2.1.0
environment: production

database:
  host: prod-db.company.com
  port: 5432
  name: myapp_prod
  ssl: true

redis:
  host: redis-cluster.company.com
  port: 6379
  db: 0

features:
  new_ui: true
  beta_features: false
  analytics: enabled

logging:
  level: info
  format: json
```

### Kubernetes Integration Pattern

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      initContainers:
      - name: config-fetcher
        image: alpine:latest
        command: ["/bin/sh", "-c"]
        args:
          - |
            apk add --no-cache curl
            curl -H "Authorization: Bearer $YAMLET_TOKEN" \
              http://yamlet-service.yamlet:8080/namespaces/$NAMESPACE/configs/app.yaml \
              -o /shared/config.yaml
        env:
        - name: YAMLET_TOKEN
          valueFrom:
            secretKeyRef:
              name: yamlet-token
              key: token
        - name: NAMESPACE
          value: "production"
        volumeMounts:
        - name: config
          mountPath: /shared
      containers:
      - name: app
        image: my-app:latest
        volumeMounts:
        - name: config
          mountPath: /config
      volumes:
      - name: config
        emptyDir: {}
```

## 🚢 Deployment Options

### Production Kubernetes

```bash
# Deploy with dedicated namespace (recommended)
kubectl apply -f https://raw.githubusercontent.com/zvdy/yamlet/main/k8s/yamlet-namespace.yaml

# Update the deployment to use Docker Hub image
kubectl set image deployment/yamlet yamlet=zvdy/yamlet:0.0.1 -n yamlet
```

### Minikube Development

```bash
# Deploy to Minikube
kubectl apply -f https://raw.githubusercontent.com/zvdy/yamlet/main/k8s/minikube.yaml

# Get service URL
minikube service yamlet-service --url
```

### Docker Compose

```yaml
version: '3.8'
services:
  yamlet:
    image: zvdy/yamlet:0.0.1
    ports:
      - "8080:8080"
    environment:
      - USE_FILES=true
      - YAMLET_ADMIN_TOKEN=your-secure-admin-token
      - YAMLET_TOKENS=prod-token:production,dev-token:development
    volumes:
      - yamlet-data:/data
    restart: unless-stopped

volumes:
  yamlet-data:
```

## 🔧 Development & Testing

### Building from Source

```bash
# Clone repository
git clone https://github.com/zvdy/yamlet.git
cd yamlet

# Install dependencies
go mod download

# Build binary
make build

# Run tests
make test

# Build Docker image
make docker-build
```

### Testing Suite

```bash
# Run unit tests
go test ./...

# Run API integration tests
./tests/test-api.sh

# Run admin token management tests
./tests/test-admin.sh

# Run complete integration tests (requires running Yamlet)
./tests/test-integration.sh
```

### Makefile Targets

```bash
make build          # Build binary
make test           # Run tests
make docker-build   # Build Docker image
make minikube-deploy # Deploy to Minikube
make minikube-test  # Run Minikube integration tests
make clean          # Clean build artifacts
```

## 🔐 Security Considerations

### Production Checklist

- [ ] **Change Admin Token**: Set `YAMLET_ADMIN_TOKEN` to a secure value
- [ ] **Use HTTPS**: Deploy behind TLS termination (ingress/load balancer)
- [ ] **Network Policies**: Restrict network access using Kubernetes NetworkPolicies
- [ ] **Resource Limits**: Set appropriate CPU/memory limits
- [ ] **Persistent Storage**: Use file storage with proper volume security
- [ ] **Secret Management**: Store tokens in Kubernetes Secrets, not ConfigMaps
- [ ] **RBAC**: Implement Kubernetes RBAC for service account permissions

### Token Security

```yaml
# Example: Storing tokens securely
apiVersion: v1
kind: Secret
metadata:
  name: yamlet-tokens
  namespace: production
type: Opaque
data:
  prod-token: <base64-encoded-token>
  yamlet-admin: <base64-encoded-admin-token>
```

## 📊 Monitoring & Observability

### Health Checks

```bash
# Basic health check
curl http://yamlet-service:8080/health

# Kubernetes probes are pre-configured in deployments
```

### Logging

Yamlet provides structured logging for:
- API requests and responses
- Authentication events
- Storage operations
- Error conditions
- Admin operations

## 🗂️ Project Structure

```
yamlet/
├── cmd/yamlet/              # Main application entry point
├── internal/
│   ├── auth/               # Authentication & token management
│   ├── handlers/           # HTTP request handlers
│   └── storage/            # Storage backend implementations
├── examples/               # Example applications & documentation
├── k8s/                   # Kubernetes deployment manifests
├── tests/                 # Integration test scripts
├── Dockerfile             # Multi-stage Docker build
├── Makefile              # Build and deployment automation
└── docs/                 # Additional documentation
```

## 🔄 Version History

### v0.0.1 (Current)
- ✅ Core REST API for YAML configuration management
- ✅ Token-based authentication with namespace isolation  
- ✅ Admin API for runtime token management
- ✅ In-memory and file-based storage backends
- ✅ Kubernetes-native deployment configurations
- ✅ Comprehensive test suite and example applications
- ✅ Docker Hub distribution

## 🛣️ Roadmap

See [ROADMAP.md](ROADMAP.md) for planned features including:
- Web UI for configuration management
- Enhanced RBAC and user management
- Configuration versioning and rollback
- Multi-node clustering and replication
- Webhook integrations
- Audit logging and compliance features

## 🤝 Contributing

We welcome contributions! Please see our contributing guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup

```bash
# Set up development environment
git clone https://github.com/zvdy/yamlet.git
cd yamlet
go mod download
make test

# Run locally with hot reload (requires air)
air
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🌟 Support

- 📚 **Documentation**: [GitHub Wiki](https://github.com/zvdy/yamlet/wiki)
- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/zvdy/yamlet/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/zvdy/yamlet/discussions)
- 🐳 **Docker Hub**: [zvdy/yamlet](https://hub.docker.com/r/zvdy/yamlet)

---

<div align="center">
  <p>Made with ❤️ for the Kubernetes community</p>
  <p>⭐ Star us on GitHub if you find Yamlet useful!</p>
</div>
