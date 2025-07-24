# Yamlet Minikube Quick Start

## 🚀 Deploy to Minikube

```bash
# One-command deploy and test
make minikube-test

# Or step by step:
make minikube-deploy    # Build and deploy
make minikube-url       # Get service URL
make minikube-logs      # Watch logs
```

## 🧪 Testing in Minikube

### Get Service URL
```bash
# Get the service URL
YAMLET_URL=$(minikube service yamlet-service --url)
echo "Yamlet is available at: $YAMLET_URL"
```

### Test the API
```bash
# Health check
curl $YAMLET_URL/health

# Store a config
curl -X POST \
  -H "Authorization: Bearer devtoken123" \
  -H "Content-Type: application/x-yaml" \
  --data-binary @example-config.yaml \
  $YAMLET_URL/namespaces/dev/configs/my-app.yaml

# Retrieve a config
curl -H "Authorization: Bearer devtoken123" \
  $YAMLET_URL/namespaces/dev/configs/my-app.yaml

# List configs
curl -H "Authorization: Bearer devtoken123" \
  $YAMLET_URL/namespaces/dev/configs
```

## 🔧 Useful Commands

### Pod Management
```bash
# Get pod status
kubectl get pods -l app=yamlet

# Watch pod logs
kubectl logs -l app=yamlet -f

# Describe pod (for troubleshooting)
kubectl describe pod -l app=yamlet

# Get into the pod
kubectl exec -it $(kubectl get pod -l app=yamlet -o jsonpath='{.items[0].metadata.name}') -- sh
```

### Port Forwarding (Alternative Access)
```bash
# Forward port 8080 to local machine
kubectl port-forward service/yamlet-service 8080:8080

# Now you can use localhost
curl http://localhost:8080/health
```

### Service Management
```bash
# Get service details
kubectl get service yamlet-service

# Get service URL (automatic)
minikube service yamlet-service --url

# Open service in browser
minikube service yamlet-service
```

## 🧹 Cleanup

```bash
# Remove Yamlet from Minikube
make minikube-clean

# Or manually
kubectl delete -f k8s/minikube.yaml
```

## 🔍 Troubleshooting

### Common Issues

1. **Image Pull Error**: Make sure you built the image in Minikube context
   ```bash
   eval $(minikube docker-env)
   docker build -t yamlet:latest .
   ```

2. **Pod not starting**: Check pod events
   ```bash
   kubectl describe pod -l app=yamlet
   ```

3. **Service not accessible**: Verify service and endpoints
   ```bash
   kubectl get service yamlet-service
   kubectl get endpoints yamlet-service
   ```

### Debug Commands
```bash
# Check all Yamlet resources
kubectl get all -l app=yamlet

# Check resource usage
kubectl top pod -l app=yamlet

# Get all events related to Yamlet
kubectl get events --field-selector involvedObject.name=yamlet
```

## 🎯 Production Considerations

When moving from Minikube to production:

1. **Use proper image registry** instead of local builds
2. **Add persistent volumes** for file-based storage
3. **Configure ingress** instead of NodePort
4. **Set resource limits** based on your needs
5. **Add monitoring** and alerting
6. **Use secrets** for tokens instead of environment variables

## 📊 Example Production Manifest

```yaml
# For production, you might want:
apiVersion: apps/v1
kind: Deployment
metadata:
  name: yamlet
spec:
  replicas: 3  # Multiple replicas
  template:
    spec:
      containers:
      - name: yamlet
        image: your-registry/yamlet:v1.0.0  # Proper versioning
        env:
        - name: USE_FILES
          value: "true"
        - name: YAMLET_TOKENS
          valueFrom:
            secretKeyRef:  # Use secrets
              name: yamlet-tokens
              key: tokens
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: yamlet-data
```
