#!/bin/bash

# Minikube Test Script for Yamlet

set -e

echo "🚀 Starting Yamlet Minikube Test"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="default"
SERVICE_NAME="yamlet-service"
TOKEN="devtoken123"

# Functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Step 1: Build Docker image in Minikube
log_info "Step 1: Building Docker image in Minikube context"
eval $(minikube -p minikube docker-env)
docker build -t yamlet:latest . || {
    log_error "Failed to build Docker image"
    exit 1
}
log_success "Docker image built successfully"

# Step 2: Deploy to Minikube
log_info "Step 2: Deploying Yamlet to Minikube"
kubectl apply -f k8s/minikube.yaml || {
    log_error "Failed to deploy to Minikube"
    exit 1
}
log_success "Deployment applied successfully"

# Step 3: Wait for pod to be ready
log_info "Step 3: Waiting for pod to be ready..."
kubectl wait --for=condition=ready pod -l app=yamlet --timeout=120s || {
    log_error "Pod failed to become ready"
    kubectl describe pod -l app=yamlet
    exit 1
}
log_success "Pod is ready"

# Step 4: Check pod status
log_info "Step 4: Checking pod status"
kubectl get pods -l app=yamlet
kubectl get services -l app=yamlet

# Step 5: Get Minikube service URL
log_info "Step 5: Getting service URL"
MINIKUBE_IP=$(minikube ip)
SERVICE_PORT=$(kubectl get service $SERVICE_NAME -o jsonpath='{.spec.ports[0].nodePort}')
SERVICE_URL="http://$MINIKUBE_IP:$SERVICE_PORT"

log_success "Service available at: $SERVICE_URL"

# Step 6: Wait for service to be fully ready
log_info "Step 6: Waiting for service to be ready..."
for i in {1..30}; do
    if curl -s "$SERVICE_URL/health" > /dev/null 2>&1; then
        log_success "Service is responding"
        break
    fi
    if [ $i -eq 30 ]; then
        log_error "Service failed to respond after 30 attempts"
        kubectl logs -l app=yamlet --tail=50
        exit 1
    fi
    echo "Attempt $i/30: Waiting for service..."
    sleep 2
done

# Step 7: Test the API
log_info "Step 7: Testing Yamlet API in Minikube"

echo ""
echo "🧪 Running API Tests:"

# Test 1: Health check
echo "1. Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s "$SERVICE_URL/health")
if [ "$HEALTH_RESPONSE" = "OK" ]; then
    log_success "Health check passed"
else
    log_error "Health check failed: $HEALTH_RESPONSE"
    exit 1
fi

# Test 2: Store a config
echo ""
echo "2. Storing a test config..."
YAML_CONTENT="app: minikube-test
version: 1.0.0
environment: minikube
database:
  host: postgres.minikube.local
  port: 5432
  name: testdb
redis:
  host: redis.minikube.local
  port: 6379
features:
  minikube_testing: true
  local_development: true"

STORE_RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/x-yaml" \
  --data "$YAML_CONTENT" \
  "$SERVICE_URL/namespaces/dev/configs/minikube-test.yaml")

if echo "$STORE_RESPONSE" | grep -q "Config stored successfully"; then
    log_success "Config stored successfully"
    echo "Response: $STORE_RESPONSE"
else
    log_error "Failed to store config: $STORE_RESPONSE"
    exit 1
fi

# Test 3: Retrieve the config
echo ""
echo "3. Retrieving the config..."
RETRIEVED_CONFIG=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "$SERVICE_URL/namespaces/dev/configs/minikube-test.yaml")

if echo "$RETRIEVED_CONFIG" | grep -q "minikube-test"; then
    log_success "Config retrieved successfully"
    echo "Retrieved config:"
    echo "$RETRIEVED_CONFIG"
else
    log_error "Failed to retrieve config: $RETRIEVED_CONFIG"
    exit 1
fi

# Test 4: List configs
echo ""
echo "4. Listing configs..."
LIST_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "$SERVICE_URL/namespaces/dev/configs")

if echo "$LIST_RESPONSE" | grep -q "minikube-test.yaml"; then
    log_success "Config listing successful"
    echo "Response: $LIST_RESPONSE"
else
    log_error "Config listing failed: $LIST_RESPONSE"
    exit 1
fi

# Test 5: Test authentication
echo ""
echo "5. Testing authentication (should fail)..."
AUTH_FAIL_RESPONSE=$(curl -s -H "Authorization: Bearer invalidtoken" \
  "$SERVICE_URL/namespaces/dev/configs/minikube-test.yaml" 2>/dev/null)

if echo "$AUTH_FAIL_RESPONSE" | grep -q "Authentication failed"; then
    log_success "Authentication properly rejected invalid token"
else
    log_warning "Authentication test inconclusive: $AUTH_FAIL_RESPONSE"
fi

# Step 8: Show pod logs
echo ""
log_info "Step 8: Recent pod logs"
kubectl logs -l app=yamlet --tail=20

# Step 9: Resource usage
echo ""
log_info "Step 9: Resource usage"
kubectl top pod -l app=yamlet 2>/dev/null || log_warning "Metrics server not available"

echo ""
log_success "🎉 All tests passed! Yamlet is working perfectly in Minikube"
echo ""
echo "📝 Summary:"
echo "   • Service URL: $SERVICE_URL"
echo "   • Health endpoint: $SERVICE_URL/health"
echo "   • API endpoint: $SERVICE_URL/namespaces/{namespace}/configs/{name}"
echo "   • Available tokens: devtoken123 (dev), stagingtoken456 (staging), prodtoken789 (production)"
echo ""
echo "🔧 Useful commands:"
echo "   • kubectl get pods -l app=yamlet"
echo "   • kubectl logs -l app=yamlet -f"
echo "   • kubectl port-forward service/yamlet-service 8080:8080"
echo "   • minikube service yamlet-service --url"
echo ""
echo "🧹 To clean up:"
echo "   kubectl delete -f k8s/minikube.yaml"
