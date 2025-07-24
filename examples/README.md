# Yamlet Examples

This directory contains example applications that demonstrate how to use Yamlet for configuration management in Kubernetes.

## Structure

```
examples/
├── mock-db/                    # Go-based mock database service
│   ├── Dockerfile             # Docker build configuration
│   ├── go.mod                 # Go module dependencies
│   ├── go.sum                 # Go module checksums
│   └── main.go                # Mock database service implementation
├── sample-app/                # Sample application demonstrating Yamlet integration
│   ├── Dockerfile             # Docker build configuration
│   ├── go.mod                 # Go module dependencies
│   ├── main.go                # Sample application implementation
│   └── yamlet-fetch-config.sh # Configuration fetcher script
└── yamlet-fetch-config.sh     # Shared configuration fetcher script
```

## Services

### Mock Database (`mock-db`)
- **Purpose**: Simulates a database service for testing
- **Port**: 3000
- **Endpoints**:
  - `GET /health` - Health check
  - `GET /db/info` - Database information
  - `GET /users` - List all users
  - `GET /users/{id}` - Get user by ID
  - `GET /config` - Current configuration

### Sample Application (`sample-app`)
- **Purpose**: Demonstrates fetching configuration from Yamlet
- **Port**: 8080
- **Features**:
  - Fetches configuration from Yamlet at startup
  - Uses database connection details from Yamlet
  - Exposes configuration endpoints for inspection
- **Endpoints**:
  - `GET /health` - Health check
  - `GET /config` - Application configuration
  - `GET /database` - Database connection test
  - `GET /` - Application information

## Deployment

The examples are deployed using the Kubernetes configuration in `../k8s/example-apps.yaml`.

### Namespaces
- `dev-apps` - Development environment applications
- `prod-apps` - Production environment applications (reserved for future use)

### Services
- `mock-db.dev-apps.svc.cluster.local:3000` - Mock database service
- `sample-app-dev.dev-apps.svc.cluster.local:8080` - Sample application

## Configuration Flow

1. **Startup**: Sample app's init container fetches configuration from Yamlet
2. **Parsing**: Configuration is parsed and converted to environment variables
3. **Loading**: Main container loads environment variables and starts the application
4. **Runtime**: Application uses configuration for database connections and features

## Testing

To test the integration:

```bash
# Test sample app health
curl http://$(minikube ip):30081/health

# View configuration loaded from Yamlet
curl http://$(minikube ip):30081/config

# Test database connection using Yamlet config
curl http://$(minikube ip):30081/database
```

## Building Images

```bash
# Build mock database
cd mock-db
eval $(minikube docker-env)
docker build -t mock-db:latest .

# Build sample app
cd ../sample-app
eval $(minikube docker-env)
docker build -t sample-app:latest .
```

## Configuration Management

The sample application demonstrates:
- **Configuration Fetching**: Using init containers to fetch config from Yamlet
- **Environment Variables**: Converting YAML to environment variables
- **Service Discovery**: Using Kubernetes service names from configuration
- **Namespace Isolation**: Different tokens for different namespaces
- **Runtime Updates**: Restarting deployments to pick up configuration changes
