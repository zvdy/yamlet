#!/bin/bash

# Test script for Yamlet API

BASE_URL="http://localhost:8080"
TOKEN="devtoken123"
NAMESPACE="dev"
CONFIG_NAME="test-app.yaml"

echo "🔍 Testing Yamlet API..."

# Test health endpoint
echo "1. Testing health endpoint..."
curl -s "$BASE_URL/health"
echo -e "\n"

# Test storing a config
echo "2. Storing a test config..."
YAML_CONTENT="app: test-app
version: 1.0.0
environment: development
database:
  host: localhost
  port: 5432
  name: testdb
features:
  feature_a: true
  feature_b: false
  feature_c: \"enabled\""

curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/x-yaml" \
  --data "$YAML_CONTENT" \
  "$BASE_URL/namespaces/$NAMESPACE/configs/$CONFIG_NAME"
echo -e "\n"

# Test retrieving the config
echo "3. Retrieving the config..."
curl -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/namespaces/$NAMESPACE/configs/$CONFIG_NAME"
echo -e "\n"

# Test listing configs
echo "4. Listing configs in namespace..."
curl -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/namespaces/$NAMESPACE/configs"
echo -e "\n"

# Test with invalid token
echo "5. Testing invalid token (should fail)..."
curl -H "Authorization: Bearer invalidtoken" \
  "$BASE_URL/namespaces/$NAMESPACE/configs/$CONFIG_NAME" 2>/dev/null
echo -e "\n"

# Test wrong namespace for token
echo "6. Testing wrong namespace (should fail)..."
curl -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/namespaces/production/configs/$CONFIG_NAME" 2>/dev/null
echo -e "\n"

echo "✅ Test complete!"
