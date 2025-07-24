#!/bin/bash

# Yamlet Configuration Fetcher Script
# This script fetches configuration from Yamlet API and sets environment variables

set -e

# Configuration
YAMLET_URL="${YAMLET_URL:-http://yamlet-service.yamlet.svc.cluster.local:8080}"
YAMLET_TOKEN="${YAMLET_TOKEN:-}"
YAMLET_NAMESPACE="${YAMLET_NAMESPACE:-dev}"
YAMLET_CONFIG="${YAMLET_CONFIG:-app.yaml}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[YAMLET] $1${NC}"
}

log_success() {
    echo -e "${GREEN}[YAMLET] $1${NC}"
}

log_error() {
    echo -e "${RED}[YAMLET] $1${NC}"
}

log_warn() {
    echo -e "${YELLOW}[YAMLET] $1${NC}"
}

# Function to fetch configuration from Yamlet
fetch_config() {
    local url="$YAMLET_URL/namespaces/$YAMLET_NAMESPACE/configs/$YAMLET_CONFIG"
    
    log_info "Fetching configuration from $url"
    
    if [ -z "$YAMLET_TOKEN" ]; then
        log_error "YAMLET_TOKEN environment variable is required"
        exit 1
    fi
    
    # Fetch the configuration
    local config_response
    config_response=$(curl -s -H "Authorization: Bearer $YAMLET_TOKEN" "$url" 2>/dev/null)
    
    if [ $? -ne 0 ] || [ -z "$config_response" ]; then
        log_error "Failed to fetch configuration from Yamlet"
        exit 1
    fi
    
    # Check if response contains an error
    if echo "$config_response" | grep -q "Authentication failed\|not found\|error"; then
        log_error "Yamlet API error: $config_response"
        exit 1
    fi
    
    log_success "Configuration fetched successfully"
    echo "$config_response"
}

# Function to parse YAML and set environment variables
parse_and_set_env() {
    local yaml_content="$1"
    local env_file="${2:-/tmp/yamlet.env}"
    
    log_info "Parsing YAML configuration and generating environment variables"
    
    # Simple YAML parser for flat key-value pairs
    # This is a basic implementation - for complex YAML, consider using yq
    echo "$yaml_content" | while IFS= read -r line; do
        # Skip empty lines and comments
        if [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]]; then
            continue
        fi
        
        # Handle simple key: value pairs
        if [[ "$line" =~ ^[[:space:]]*([^:]+):[[:space:]]*(.*)$ ]]; then
            key="${BASH_REMATCH[1]}"
            value="${BASH_REMATCH[2]}"
            
            # Clean up key and value
            key=$(echo "$key" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | tr '[:lower:]' '[:upper:]' | tr '.' '_' | tr '-' '_')
            value=$(echo "$value" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | sed 's/^["'\'']*//;s/["'\'']*$//')
            
            # Skip nested objects (basic detection)
            if [[ ! "$value" =~ ^[[:space:]]*$ && ! "$line" =~ ^[[:space:]]*[^:]+:[[:space:]]*$ ]]; then
                echo "export ${key}=\"${value}\"" >> "$env_file"
                log_info "Set ${key}=${value}"
            fi
        fi
    done
    
    log_success "Environment variables written to $env_file"
}

# Function to export environment variables from file
source_env_file() {
    local env_file="${1:-/tmp/yamlet.env}"
    
    if [ -f "$env_file" ]; then
        log_info "Sourcing environment variables from $env_file"
        source "$env_file"
        log_success "Environment variables loaded"
    else
        log_warn "Environment file $env_file not found"
    fi
}

# Main execution
main() {
    log_info "Starting Yamlet configuration fetch"
    log_info "Namespace: $YAMLET_NAMESPACE"
    log_info "Config: $YAMLET_CONFIG"
    log_info "URL: $YAMLET_URL"
    
    # Fetch configuration
    local config
    config=$(fetch_config)
    
    # Parse and set environment variables
    local env_file="/tmp/yamlet.env"
    rm -f "$env_file"
    parse_and_set_env "$config" "$env_file"
    
    # If running as source, export variables to current shell
    if [ "$1" = "--source" ]; then
        source_env_file "$env_file"
    fi
    
    log_success "Yamlet configuration fetch completed"
    
    # Display loaded environment variables
    if [ -f "$env_file" ]; then
        log_info "Loaded environment variables:"
        cat "$env_file" | sed 's/export /  /' | sed 's/=/ = /'
    fi
}

# Run main function
main "$@"
