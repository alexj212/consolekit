# Production Server Example

This example demonstrates a production-ready deployment of ConsoleKit with:
- Environment-based configuration
- Multiple transports (HTTP, SSH, REPL)
- Command logging and audit trails
- Graceful shutdown
- Security best practices
- Docker support
- Systemd service files

## Features

- **Environment Configuration**: All settings via environment variables
- **Multiple Transports**: Enable/disable HTTP, SSH, and REPL independently
- **Security**: Command logging, password authentication, filtered commands
- **Monitoring**: Health checks, status endpoint, metrics
- **Production Ready**: Graceful shutdown, error handling, logging
- **Docker Support**: Dockerfile and docker-compose.yml included
- **Systemd Integration**: Service file for Linux deployments

## Quick Start

### Local Development

```bash
cd examples/production_server
go build

# Run with defaults
./production-server

# Run with custom config
HTTP_PORT=9090 SSH_PORT=2223 ./production-server
```

### Docker

```bash
# Build image
docker build -t consolekit-server .

# Run container
docker run -p 8080:8080 -p 2222:2222 \
  -e HTTP_PASSWORD=mysecret \
  -e SSH_PASSWORD=sshsecret \
  consolekit-server
```

### Docker Compose

```bash
docker-compose up -d
```

## Configuration

All configuration via environment variables:

### Server Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `8080` | HTTP server port |
| `HTTP_USER` | `admin` | HTTP username |
| `HTTP_PASSWORD` | `changeme` | HTTP password |
| `SSH_PORT` | `2222` | SSH server port |
| `SSH_PASSWORD` | `changeme` | SSH password |

### Feature Flags

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_HTTP` | `true` | Enable HTTP/WebSocket server |
| `ENABLE_SSH` | `true` | Enable SSH server |
| `ENABLE_REPL` | `false` | Enable local REPL (for debugging) |

### Security Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_COMMANDS` | `true` | Enable command logging |
| `LOG_FILE` | `/var/log/consolekit-commands.log` | Log file path |
| `TS_AUTH_KEY` | - | Tailscale auth key (optional) |

### Example Configuration

```bash
# Production settings
export HTTP_PORT=8080
export HTTP_USER=admin
export HTTP_PASSWORD=SuperSecretPassword123!
export SSH_PORT=2222
export SSH_PASSWORD=SSHSecretPassword456!
export LOG_COMMANDS=true
export LOG_FILE=/var/log/consolekit/commands.log
export ENABLE_HTTP=true
export ENABLE_SSH=true
export ENABLE_REPL=false

./production-server
```

## Deployment

### Systemd Service

Create `/etc/systemd/system/consolekit.service`:

```ini
[Unit]
Description=ConsoleKit Production Server
After=network.target

[Service]
Type=simple
User=consolekit
Group=consolekit
WorkingDirectory=/opt/consolekit
ExecStart=/opt/consolekit/production-server
Restart=always
RestartSec=10

# Environment
Environment="HTTP_PORT=8080"
Environment="HTTP_USER=admin"
Environment="HTTP_PASSWORD=changeme"
Environment="SSH_PORT=2222"
Environment="SSH_PASSWORD=changeme"
Environment="LOG_COMMANDS=true"
Environment="LOG_FILE=/var/log/consolekit/commands.log"

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/consolekit

[Install]
WantedBy=multi-user.target
```

**Install and start:**

```bash
# Create user
sudo useradd -r -s /bin/false consolekit

# Create directories
sudo mkdir -p /opt/consolekit /var/log/consolekit
sudo chown consolekit:consolekit /var/log/consolekit

# Copy binary
sudo cp production-server /opt/consolekit/
sudo chown consolekit:consolekit /opt/consolekit/production-server

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable consolekit
sudo systemctl start consolekit

# Check status
sudo systemctl status consolekit

# View logs
sudo journalctl -u consolekit -f
```

### Docker Deployment

**Dockerfile:**

```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY . .
RUN go build -o production-server \
    -ldflags "-X main.Version=1.0.0 -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /build/production-server .

# Create non-root user
RUN addgroup -g 1000 consolekit && \
    adduser -D -u 1000 -G consolekit consolekit && \
    mkdir -p /var/log/consolekit && \
    chown -R consolekit:consolekit /var/log/consolekit

USER consolekit

EXPOSE 8080 2222

CMD ["./production-server"]
```

**docker-compose.yml:**

```yaml
version: '3.8'

services:
  consolekit:
    build: .
    image: consolekit-server:latest
    ports:
      - "8080:8080"
      - "2222:2222"
    environment:
      - HTTP_PORT=8080
      - HTTP_USER=admin
      - HTTP_PASSWORD=${HTTP_PASSWORD:-changeme}
      - SSH_PORT=2222
      - SSH_PASSWORD=${SSH_PASSWORD:-changeme}
      - LOG_COMMANDS=true
      - ENABLE_HTTP=true
      - ENABLE_SSH=true
      - ENABLE_REPL=false
    volumes:
      - consolekit-logs:/var/log/consolekit
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

volumes:
  consolekit-logs:
```

**Run with Docker Compose:**

```bash
# Set passwords
echo "HTTP_PASSWORD=supersecret" > .env
echo "SSH_PASSWORD=sshsecret" >> .env

# Start services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Kubernetes Deployment

**deployment.yaml:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: consolekit
  labels:
    app: consolekit
spec:
  replicas: 2
  selector:
    matchLabels:
      app: consolekit
  template:
    metadata:
      labels:
        app: consolekit
    spec:
      containers:
      - name: consolekit
        image: consolekit-server:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 2222
          name: ssh
        env:
        - name: HTTP_PORT
          value: "8080"
        - name: HTTP_USER
          valueFrom:
            secretKeyRef:
              name: consolekit-secret
              key: http-user
        - name: HTTP_PASSWORD
          valueFrom:
            secretKeyRef:
              name: consolekit-secret
              key: http-password
        - name: SSH_PORT
          value: "2222"
        - name: SSH_PASSWORD
          valueFrom:
            secretKeyRef:
              name: consolekit-secret
              key: ssh-password
        - name: LOG_COMMANDS
          value: "true"
        - name: ENABLE_HTTP
          value: "true"
        - name: ENABLE_SSH
          value: "true"
        - name: ENABLE_REPL
          value: "false"
        resources:
          limits:
            memory: "256Mi"
            cpu: "500m"
          requests:
            memory: "128Mi"
            cpu: "250m"
        livenessProbe:
          httpGet:
            path: /api/v1/health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /api/v1/health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        volumeMounts:
        - name: logs
          mountPath: /var/log/consolekit
      volumes:
      - name: logs
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: consolekit
spec:
  selector:
    app: consolekit
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 8080
  - name: ssh
    protocol: TCP
    port: 2222
    targetPort: 2222
  type: LoadBalancer
---
apiVersion: v1
kind: Secret
metadata:
  name: consolekit-secret
type: Opaque
stringData:
  http-user: admin
  http-password: changeme
  ssh-password: changeme
```

**Deploy:**

```bash
kubectl apply -f deployment.yaml
kubectl get pods -l app=consolekit
kubectl logs -f deployment/consolekit
```

## Security Best Practices

### 1. Strong Passwords

```bash
# Generate strong passwords
openssl rand -base64 32

# Set in environment
export HTTP_PASSWORD=$(openssl rand -base64 32)
export SSH_PASSWORD=$(openssl rand -base64 32)
```

### 2. HTTPS Only (Production)

Use a reverse proxy like Nginx or Caddy:

**Nginx configuration:**

```nginx
server {
    listen 443 ssl http2;
    server_name console.example.com;

    ssl_certificate /etc/letsencrypt/live/console.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/console.example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

**Caddy configuration:**

```
console.example.com {
    reverse_proxy localhost:8080
}
```

### 3. Firewall Rules

```bash
# Allow only specific IPs for SSH
sudo ufw allow from 192.168.1.0/24 to any port 2222

# Allow HTTPS from anywhere
sudo ufw allow 443

# Enable firewall
sudo ufw enable
```

### 4. Command Filtering

Edit `main.go` to restrict commands:

```go
config := &consolekit.TransportConfig{
    Executor: executor,
    DeniedCommands: []string{
        "osexec",  // Disable OS command execution
        "clip",
        "paste",
    },
}
sshHandler.SetTransportConfig(config)
```

### 5. Tailscale for Secure Access

```bash
# Get Tailscale auth key from https://login.tailscale.com/admin/settings/keys

# Run with Tailscale
TS_AUTH_KEY=tskey-auth-xxx ./production-server
```

## Monitoring

### Health Checks

```bash
# HTTP health check
curl http://localhost:8080/api/v1/health

# System info
curl http://localhost:8080/api/v1/info
```

### Logging

View command audit log:

```bash
tail -f /var/log/consolekit/commands.log
```

**Log format:**

```json
{
  "timestamp": "2026-01-31T03:00:00Z",
  "user": "admin",
  "command": "print 'hello'",
  "output": "hello\n",
  "duration_ms": 5.2,
  "success": true
}
```

### Metrics (Optional)

Add Prometheus metrics:

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

router.Handle("/metrics", promhttp.Handler())
```

## Troubleshooting

### Service Won't Start

```bash
# Check logs
sudo journalctl -u consolekit -n 50

# Check config
sudo systemctl cat consolekit

# Test binary directly
sudo -u consolekit /opt/consolekit/production-server
```

### Permission Denied on Log File

```bash
# Check permissions
ls -la /var/log/consolekit/

# Fix ownership
sudo chown -R consolekit:consolekit /var/log/consolekit/

# Fix permissions
sudo chmod 755 /var/log/consolekit/
```

### Port Already in Use

```bash
# Check what's using the port
sudo lsof -i :8080

# Change port
HTTP_PORT=9090 ./production-server
```

## Backup and Recovery

### Backup Configuration

```bash
# Backup logs
tar -czf consolekit-logs-$(date +%Y%m%d).tar.gz /var/log/consolekit/

# Backup config (if using file-based config)
cp ~/.consolekit/config.toml ~/.consolekit/config.toml.backup
```

### Disaster Recovery

1. Reinstall application
2. Restore configuration files
3. Restore log files (for audit trail)
4. Restart service

## Performance Tuning

### Resource Limits (systemd)

```ini
[Service]
LimitNOFILE=65536
MemoryLimit=512M
CPUQuota=200%
```

### Docker Resource Limits

```yaml
services:
  consolekit:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 512M
        reservations:
          cpus: '1.0'
          memory: 256M
```

## See Also

- [REST API Example](../rest_api/) - REST API integration
- [SSH Server Example](../ssh_server/) - SSH-only deployment
- [Tailscale Example](../tailscale_http/) - Tailscale integration
- [Multi-Transport Example](../multi_transport/) - All transports
