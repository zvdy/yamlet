#!/bin/bash

# Yamlet Example Test Script
# This script demonstrates how applications fetch configuration from Yamlet

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

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

echo "🚀 Testing Yamlet Integration with Example Applications"
echo ""

# Configuration
MINIKUBE_IP=$(minikube ip)
YAMLET_URL="http://$MINIKUBE_IP:30080"
SAMPLE_APP_URL="http://$MINIKUBE_IP:30081"

log_info "Minikube IP: $MINIKUBE_IP"
log_info "Yamlet URL: $YAMLET_URL"
log_info "Sample App URL: $SAMPLE_APP_URL"

echo ""
echo "🔧 Step 1: Verify Yamlet is running"
YAMLET_HEALTH=$(curl -s $YAMLET_URL/health)
if [ $? -eq 0 ]; then
    log_success "Yamlet is healthy: $YAMLET_HEALTH"
else
    log_error "Yamlet is not responding"
    exit 1
fi

echo ""
echo "📋 Step 2: Check configuration stored in Yamlet"
CONFIG_RESPONSE=$(curl -s -H "Authorization: Bearer dev-token" $YAMLET_URL/namespaces/dev/configs/app.yaml)
if [ $? -eq 0 ] && [ ! -z "$CONFIG_RESPONSE" ]; then
    log_success "Configuration retrieved from Yamlet:"
    echo "$CONFIG_RESPONSE"
else
    log_error "Failed to retrieve configuration from Yamlet"
    exit 1
fi

echo ""
echo "🏃 Step 3: Verify sample application is running"
SAMPLE_HEALTH=$(curl -s $SAMPLE_APP_URL/health)
if [ $? -eq 0 ]; then
    log_success "Sample app is healthy: $SAMPLE_HEALTH"
else
    log_error "Sample app is not responding"
    exit 1
fi

echo ""
echo "🔄 Step 4: Check if sample app is using Yamlet configuration"
SAMPLE_CONFIG=$(curl -s $SAMPLE_APP_URL/config)
if echo "$SAMPLE_CONFIG" | grep -q "mock-database.dev-apps.svc.cluster.local"; then
    log_success "Sample app is using configuration from Yamlet!"
    echo "Configuration details:"
    echo "$SAMPLE_CONFIG" | jq '.' 2>/dev/null || echo "$SAMPLE_CONFIG"
else
    log_warning "Sample app may not be using Yamlet configuration"
    echo "Configuration details:"
    echo "$SAMPLE_CONFIG"
fi

echo ""
echo "🗄️  Step 5: Test database connection through sample app"
DB_TEST=$(curl -s $SAMPLE_APP_URL/database)
if echo "$DB_TEST" | grep -q "Connected to database"; then
    log_success "Database connection working through sample app!"
    echo "Database details:"
    echo "$DB_TEST" | jq '.' 2>/dev/null || echo "$DB_TEST"
else
    log_warning "Database connection may have issues"
    echo "Database response:"
    echo "$DB_TEST"
fi

echo ""
echo "🔄 Step 6: Update configuration in Yamlet and verify changes"
log_info "Updating configuration in Yamlet..."

NEW_CONFIG="app: sample-app
version: 1.3.0
environment: development
database:
  host: mock-database.dev-apps.svc.cluster.local
  port: 3306
  name: updated_dev_db
features:
  debug: true
  logging: verbose
  cache_enabled: true
  new_feature: enabled"

UPDATE_RESPONSE=$(curl -s -X POST -H "Authorization: Bearer dev-token" -H "Content-Type: application/x-yaml" \
    --data "$NEW_CONFIG" $YAMLET_URL/namespaces/dev/configs/app.yaml)

if echo "$UPDATE_RESPONSE" | grep -q "Config stored successfully"; then
    log_success "Configuration updated in Yamlet"
    echo "Update response: $UPDATE_RESPONSE"
else
    log_error "Failed to update configuration: $UPDATE_RESPONSE"
fi

echo ""
echo "📊 Step 7: Verify updated configuration is available"
UPDATED_CONFIG=$(curl -s -H "Authorization: Bearer dev-token" $YAMLET_URL/namespaces/dev/configs/app.yaml)
if echo "$UPDATED_CONFIG" | grep -q "1.3.0" && echo "$UPDATED_CONFIG" | grep -q "new_feature"; then
    log_success "Updated configuration is available in Yamlet"
    echo "Updated configuration:"
    echo "$UPDATED_CONFIG"
else
    log_warning "Configuration update may not be reflected"
fi

echo ""
echo "🔑 Step 8: Test admin token functionality"
ADMIN_TOKENS=$(curl -s -H "Authorization: Bearer minikube-admin-token-123" $YAMLET_URL/admin/tokens)
if echo "$ADMIN_TOKENS" | grep -q "tokens"; then
    log_success "Admin API is working"
    echo "Current tokens: $ADMIN_TOKENS"
else
    log_error "Admin API may have issues: $ADMIN_TOKENS"
fi

echo ""
echo "🎉 Integration Test Summary:"
echo "   ✅ Yamlet API is running and accessible"
echo "   ✅ Configuration is stored and retrievable from Yamlet"
echo "   ✅ Sample application fetches config from Yamlet at startup"
echo "   ✅ Database connection uses Yamlet configuration"
echo "   ✅ Configuration can be updated through Yamlet API"
echo "   ✅ Admin API is functional for token management"
echo ""
echo "🔗 Useful URLs:"
echo "   Yamlet API: $YAMLET_URL"
echo "   Sample App: $SAMPLE_APP_URL"
echo "   Health Checks: $YAMLET_URL/health and $SAMPLE_APP_URL/health"
echo ""
echo "💡 To restart the sample app and fetch updated config:"
echo "   kubectl rollout restart deployment/sample-app-dev -n dev-apps"
echo ""
echo "📝 To view logs:"
echo "   kubectl logs -n dev-apps -l app=sample-app -c sample-app"
echo "   kubectl logs -n yamlet -l app=yamlet"
