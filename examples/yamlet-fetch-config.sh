#!/bin/bash

# Yamlet Configuration Fetcher
# This script fetches configuration from Yamlet API and sets environment variables

set -e

# Configuration
YAMLET_URL="${YAMLET_URL:-http://yamlet-service.yamlet.svc.cluster.local:8080}"
YAMLET_TOKEN="${YAMLET_TOKEN:-}"
YAMLET_NAMESPACE="${YAMLET_NAMESPACE:-dev}"
CONFIG_NAME="${CONFIG_NAME:-app.yaml}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Validate required environment variables
if [ -z "$YAMLET_TOKEN" ]; then
    log_error "YAMLET_TOKEN environment variable is required"
    exit 1
fi

log_info "Fetching configuration from Yamlet..."
log_info "URL: $YAMLET_URL"
log_info "Namespace: $YAMLET_NAMESPACE"
log_info "Config: $CONFIG_NAME"

# Fetch configuration from Yamlet
CONFIG_RESPONSE=$(curl -s -H "Authorization: Bearer $YAMLET_TOKEN" \
    "$YAMLET_URL/namespaces/$YAMLET_NAMESPACE/configs/$CONFIG_NAME" 2>/dev/null)

if [ $? -ne 0 ] || [ -z "$CONFIG_RESPONSE" ]; then
    log_error "Failed to fetch configuration from Yamlet"
    exit 1
fi

log_success "Configuration fetched successfully"

# Parse YAML and convert to environment variables
# This is a simple parser - in production you'd use a proper YAML parser
echo "$CONFIG_RESPONSE" > /tmp/config.yaml

# Create environment file
ENV_FILE="/tmp/yamlet.env"
echo "# Generated from Yamlet configuration" > $ENV_FILE
echo "# Namespace: $YAMLET_NAMESPACE" >> $ENV_FILE
echo "# Config: $CONFIG_NAME" >> $ENV_FILE
echo "# Timestamp: $(date)" >> $ENV_FILE
echo "" >> $ENV_FILE

# Simple YAML to env conversion (basic key-value pairs)
while IFS=':' read -r key value; do
    if [[ $key =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]] && [[ ! $key =~ ^[[:space:]]*# ]]; then
        # Clean up the value
        clean_value=$(echo "$value" | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//' | sed 's/^"//' | sed 's/"$//')
        if [ ! -z "$clean_value" ]; then
            echo "${key^^}=$clean_value" >> $ENV_FILE
        fi
    fi
done < /tmp/config.yaml

# Handle nested values (simple dot notation)
if grep -q "database:" /tmp/config.yaml; then
    db_host=$(grep -A 10 "database:" /tmp/config.yaml | grep "host:" | cut -d':' -f2 | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//')
    db_port=$(grep -A 10 "database:" /tmp/config.yaml | grep "port:" | cut -d':' -f2 | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//')
    db_name=$(grep -A 10 "database:" /tmp/config.yaml | grep "name:" | cut -d':' -f2 | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//')
    
    if [ ! -z "$db_host" ]; then echo "DATABASE_HOST=$db_host" >> $ENV_FILE; fi
    if [ ! -z "$db_port" ]; then echo "DATABASE_PORT=$db_port" >> $ENV_FILE; fi
    if [ ! -z "$db_name" ]; then echo "DATABASE_NAME=$db_name" >> $ENV_FILE; fi
fi

log_success "Environment file created: $ENV_FILE"
echo ""
echo "Generated environment variables:"
cat $ENV_FILE | grep -v "^#" | grep -v "^$"

# Source the environment file if requested
if [ "$1" = "source" ]; then
    set -a
    source $ENV_FILE
    set +a
    log_success "Environment variables loaded into current shell"
fi

# Execute command with environment variables if provided
if [ "$1" = "exec" ] && [ $# -gt 1 ]; then
    shift
    log_info "Executing command with Yamlet configuration: $@"
    set -a
    source $ENV_FILE
    set +a
    exec "$@"
fi
