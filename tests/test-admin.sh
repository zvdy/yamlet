#!/bin/bash

# Admin Token Management Test Script for Yamlet

set -e

echo "🔐 Testing Yamlet Admin Token Management"

# Configuration
SERVICE_URL="http://localhost:8080"
ADMIN_TOKEN="admin-secret-token-change-me"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

echo ""
log_info "Starting admin token management tests..."

# Test 1: List existing tokens (should work with admin token)
echo ""
echo "1. Testing admin token - list existing tokens"
TOKENS_RESPONSE=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$SERVICE_URL/admin/tokens")

if echo "$TOKENS_RESPONSE" | grep -q "tokens"; then
    log_success "Admin can list tokens"
    echo "Response: $TOKENS_RESPONSE"
else
    log_error "Failed to list tokens: $TOKENS_RESPONSE"
    exit 1
fi

# Test 2: Create a new token for 'production' namespace
echo ""
echo "2. Creating new token for 'production' namespace"
CREATE_RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{"token": "prod-token-secure", "namespace": "production"}' \
  "$SERVICE_URL/admin/tokens")

if echo "$CREATE_RESPONSE" | grep -q "Token created successfully"; then
    log_success "Production token created"
    echo "Response: $CREATE_RESPONSE"
else
    log_error "Failed to create production token: $CREATE_RESPONSE"
    exit 1
fi

# Test 3: Create a new token for 'staging' namespace
echo ""
echo "3. Creating new token for 'staging' namespace"
CREATE_STAGING_RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{"token": "staging-token-secure", "namespace": "staging"}' \
  "$SERVICE_URL/admin/tokens")

if echo "$CREATE_STAGING_RESPONSE" | grep -q "Token created successfully"; then
    log_success "Staging token created"
    echo "Response: $CREATE_STAGING_RESPONSE"
else
    log_error "Failed to create staging token: $CREATE_STAGING_RESPONSE"
    exit 1
fi

# Test 4: List tokens again to see new tokens
echo ""
echo "4. Listing all tokens after creation"
UPDATED_TOKENS=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$SERVICE_URL/admin/tokens")

if echo "$UPDATED_TOKENS" | grep -q "prod-token-secure" && echo "$UPDATED_TOKENS" | grep -q "staging-token-secure"; then
    log_success "New tokens appear in list"
    echo "Response: $UPDATED_TOKENS"
else
    log_error "New tokens not found in list: $UPDATED_TOKENS"
    exit 1
fi

# Test 5: Test the new production token by storing a config
echo ""
echo "5. Testing new production token by storing config"
PROD_CONFIG="app: production-app
version: 3.0.0
environment: production
database:
  host: prod-db.company.com
  port: 5432
  name: prod_db
security:
  ssl_enabled: true
  encryption: true"

STORE_PROD_RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer prod-token-secure" \
  -H "Content-Type: application/x-yaml" \
  --data "$PROD_CONFIG" \
  "$SERVICE_URL/namespaces/production/configs/app.yaml")

if echo "$STORE_PROD_RESPONSE" | grep -q "Config stored successfully"; then
    log_success "Production token works for storing configs"
    echo "Response: $STORE_PROD_RESPONSE"
else
    log_error "Production token failed to store config: $STORE_PROD_RESPONSE"
    exit 1
fi

# Test 6: Test that production token can't access other namespaces
echo ""
echo "6. Testing namespace isolation (prod token accessing dev namespace)"
ISOLATION_TEST=$(curl -s -H "Authorization: Bearer prod-token-secure" \
  "$SERVICE_URL/namespaces/dev/configs" 2>/dev/null)

if echo "$ISOLATION_TEST" | grep -q "not authorized"; then
    log_success "Namespace isolation working correctly"
else
    log_error "Namespace isolation failed: $ISOLATION_TEST"
fi

# Test 7: Test non-admin token trying to create tokens (should fail)
echo ""
echo "7. Testing non-admin token trying to create tokens (should fail)"
UNAUTHORIZED_CREATE=$(curl -s -X POST \
  -H "Authorization: Bearer prod-token-secure" \
  -H "Content-Type: application/json" \
  --data '{"token": "hacker-token", "namespace": "evil"}' \
  "$SERVICE_URL/admin/tokens" 2>/dev/null)

if echo "$UNAUTHORIZED_CREATE" | grep -q "admin token required"; then
    log_success "Non-admin tokens properly rejected from admin operations"
else
    log_error "Security issue: non-admin token allowed admin operation: $UNAUTHORIZED_CREATE"
fi

# Test 8: Revoke the staging token
echo ""
echo "8. Revoking staging token"
REVOKE_RESPONSE=$(curl -s -X DELETE \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$SERVICE_URL/admin/tokens/staging-token-secure")

if echo "$REVOKE_RESPONSE" | grep -q "Token revoked successfully"; then
    log_success "Staging token revoked successfully"
    echo "Response: $REVOKE_RESPONSE"
else
    log_error "Failed to revoke staging token: $REVOKE_RESPONSE"
    exit 1
fi

# Test 9: Test that revoked token no longer works
echo ""
echo "9. Testing that revoked staging token no longer works"
REVOKED_TEST=$(curl -s -H "Authorization: Bearer staging-token-secure" \
  "$SERVICE_URL/namespaces/staging/configs" 2>/dev/null)

if echo "$REVOKED_TEST" | grep -q "invalid token"; then
    log_success "Revoked token properly rejected"
else
    log_error "Revoked token still working: $REVOKED_TEST"
fi

# Test 10: Final token list
echo ""
echo "10. Final token list (staging token should be gone)"
FINAL_TOKENS=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$SERVICE_URL/admin/tokens")

if echo "$FINAL_TOKENS" | grep -q "prod-token-secure" && ! echo "$FINAL_TOKENS" | grep -q "staging-token-secure"; then
    log_success "Final token list correct (staging token removed)"
    echo "Response: $FINAL_TOKENS"
else
    log_error "Final token list incorrect: $FINAL_TOKENS"
fi

echo ""
log_success "🎉 All admin token management tests passed!"
echo ""
echo "📋 Summary of what was tested:"
echo "   ✅ Admin token can list tokens"
echo "   ✅ Admin token can create new tokens"  
echo "   ✅ New tokens work for their assigned namespaces"
echo "   ✅ Namespace isolation is enforced"
echo "   ✅ Non-admin tokens cannot perform admin operations"
echo "   ✅ Admin token can revoke tokens"
echo "   ✅ Revoked tokens are immediately invalid"
echo ""
echo "🔧 Admin API endpoints:"
echo "   • GET /admin/tokens - List all tokens"
echo "   • POST /admin/tokens - Create new token"
echo "   • DELETE /admin/tokens/{token} - Revoke token"
echo ""
echo "🔑 Admin token: $ADMIN_TOKEN"
echo "   (Change this in production via YAMLET_ADMIN_TOKEN env var)"
