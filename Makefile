# Yamlet Makefile

# Variables
BINARY_NAME=yamlet
DOCKER_IMAGE=yamlet
DOCKER_TAG=latest
GOARCH=amd64
GOOS=linux

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

.PHONY: all build clean test deps docker-build docker-run k8s-deploy k8s-delete help

# Default target
all: deps test build

# Build the binary
build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -a -installsuffix cgo -ldflags '-extldflags "-static"' -o $(BINARY_NAME) ./cmd/yamlet

# Build for local development
build-local:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/yamlet

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Run tests
test:
	$(GOTEST) -v ./...

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run locally
run: build-local
	./$(BINARY_NAME)

# Run with file storage
run-files: build-local
	./$(BINARY_NAME) -use-files -data-dir ./data

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Run Docker container
docker-run: docker-build
	docker run -p 8080:8080 -v $(PWD)/data:/data $(DOCKER_IMAGE):$(DOCKER_TAG)

# Deploy to Kubernetes
k8s-deploy: docker-build
	kubectl apply -f k8s/yamlet.yaml

# Delete from Kubernetes
k8s-delete:
	kubectl delete -f k8s/yamlet.yaml

# Load Docker image into k3s
k3s-load: docker-build
	k3s ctr images import <(docker save $(DOCKER_IMAGE):$(DOCKER_TAG))

# Minikube targets
minikube-build: 
	eval $$(minikube docker-env) && docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

minikube-deploy: minikube-build
	kubectl apply -f k8s/minikube.yaml

minikube-test: 
	./test-minikube.sh

minikube-clean:
	kubectl delete -f k8s/minikube.yaml --ignore-not-found=true

minikube-logs:
	kubectl logs -l app=yamlet -f

minikube-url:
	@echo "Service URL: http://$$(minikube ip):$$(kubectl get service yamlet-service -o jsonpath='{.spec.ports[0].nodePort}')"

# Quick test endpoints
test-api:
	@echo "Testing health endpoint..."
	curl -s http://localhost:8080/health
	@echo "\n\nStoring test config..."
	echo "app: test\nversion: 1.0" | curl -X POST -H "Authorization: Bearer devtoken123" -H "Content-Type: application/x-yaml" --data-binary @- http://localhost:8080/namespaces/dev/configs/test.yaml
	@echo "\n\nRetrieving test config..."
	curl -H "Authorization: Bearer devtoken123" http://localhost:8080/namespaces/dev/configs/test.yaml
	@echo "\n\nListing configs..."
	curl -H "Authorization: Bearer devtoken123" http://localhost:8080/namespaces/dev/configs

# Format code
fmt:
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Security scan (requires gosec)
security:
	gosec ./...

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary for Linux"
	@echo "  build-local  - Build the binary for local development"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  run          - Build and run locally"
	@echo "  run-files    - Build and run with file storage"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Build and run Docker container"
	@echo "  k8s-deploy   - Deploy to Kubernetes"
	@echo "  k8s-delete   - Delete from Kubernetes"
	@echo "  k3s-load     - Load Docker image into k3s"
	@echo "  minikube-build - Build Docker image in Minikube"
	@echo "  minikube-deploy - Deploy to Minikube"
	@echo "  minikube-test - Run full Minikube test suite"
	@echo "  minikube-clean - Remove from Minikube"
	@echo "  minikube-logs - Follow pod logs in Minikube"
	@echo "  minikube-url - Show Minikube service URL"
	@echo "  test-api     - Test API endpoints"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  security     - Security scan"
	@echo "  help         - Show this help"
