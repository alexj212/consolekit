# REST API Example

This example demonstrates how to build a REST API on top of ConsoleKit, combining:
- HTTP REST API for programmatic access
- WebSocket REPL for interactive web terminal
- Background job management
- Variable management
- Health monitoring

## Features

- **Command Execution API**: Execute ConsoleKit commands via HTTP POST
- **Job Management API**: List, view, and kill background jobs
- **Variable Management API**: CRUD operations for variables
- **Health Monitoring**: Health check and system info endpoints
- **Web Terminal**: Full xterm.js terminal at `/admin`
- **JSON Responses**: All API responses in JSON format
- **Error Handling**: Proper HTTP status codes and error messages

## Building

```bash
cd examples/rest_api
go build
```

## Running

```bash
./rest_api
```

Server starts on http://localhost:8080

## API Endpoints

### Command Execution

#### Execute Command
```bash
POST /api/v1/execute
Content-Type: application/json

{
  "command": "print \"Hello from API\"",
  "timeout": 30,
  "env": {
    "USER": "api-user",
    "KEY": "value"
  }
}
```

**Response:**
```json
{
  "output": "Hello from API\n",
  "duration_ms": 5.2,
  "success": true
}
```

**Examples:**
```bash
# Simple command
curl -X POST http://localhost:8080/api/v1/execute \
  -H 'Content-Type: application/json' \
  -d '{"command": "date"}'

# With timeout
curl -X POST http://localhost:8080/api/v1/execute \
  -H 'Content-Type: application/json' \
  -d '{"command": "sleep 5s", "timeout": 10}'

# With environment variables
curl -X POST http://localhost:8080/api/v1/execute \
  -H 'Content-Type: application/json' \
  -d '{
    "command": "print @NAME",
    "env": {"NAME": "Alice"}
  }'
```

### Job Management

#### List All Jobs
```bash
GET /api/v1/jobs
```

**Response:**
```json
{
  "jobs": [
    {
      "id": "job-1",
      "command": "sleep 100",
      "status": "running",
      "pid": 12345,
      "started": "2026-01-31T03:00:00Z"
    }
  ],
  "count": 1
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/jobs
```

#### Get Job Details
```bash
GET /api/v1/jobs/{id}
```

**Response:**
```json
{
  "id": "job-1",
  "command": "sleep 100",
  "status": "running",
  "pid": 12345,
  "started": "2026-01-31T03:00:00Z",
  "finished": null,
  "output": ""
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/jobs/job-1
```

#### Kill Job
```bash
POST /api/v1/jobs/{id}/kill
```

**Response:**
```json
{
  "success": true,
  "message": "Job killed"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/jobs/job-1/kill
```

### Variable Management

#### List All Variables
```bash
GET /api/v1/variables
```

**Response:**
```json
{
  "variables": {
    "@user": "admin",
    "@env": "production"
  },
  "count": 2
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/variables
```

#### Get Variable
```bash
GET /api/v1/variables/{name}
```

**Response:**
```json
{
  "name": "user",
  "value": "admin"
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/variables/user
```

#### Set Variable
```bash
PUT /api/v1/variables/{name}
Content-Type: application/json

{
  "value": "new-value"
}
```

**Response:**
```json
{
  "success": true,
  "name": "user",
  "value": "new-value"
}
```

**Example:**
```bash
curl -X PUT http://localhost:8080/api/v1/variables/user \
  -H 'Content-Type: application/json' \
  -d '{"value": "alice"}'
```

#### Delete Variable
```bash
DELETE /api/v1/variables/{name}
```

**Response:**
```json
{
  "success": true,
  "message": "Variable deleted"
}
```

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/v1/variables/user
```

### Health & System Info

#### Health Check
```bash
GET /api/v1/health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "v1.0.0",
  "uptime": "2h15m30s"
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/health
```

#### System Information
```bash
GET /api/v1/info
```

**Response:**
```json
{
  "app_name": "rest-api",
  "version": "v1.0.0",
  "build_date": "2026-01-31T03:00:00Z",
  "commit": "abc123",
  "uptime": "2h15m30s",
  "jobs_total": 5,
  "jobs_running": 2,
  "variables": 10,
  "logging": true,
  "max_recursion": 10
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/info
```

## Web Terminal

Access the interactive web terminal at:
```
http://localhost:8080/admin
```

**Login:**
- Username: `admin`
- Password: `secret123`

## Complete Workflow Example

### 1. Check System Health
```bash
curl http://localhost:8080/api/v1/health
```

### 2. Set Variables
```bash
curl -X PUT http://localhost:8080/api/v1/variables/environment \
  -H 'Content-Type: application/json' \
  -d '{"value": "production"}'

curl -X PUT http://localhost:8080/api/v1/variables/region \
  -H 'Content-Type: application/json' \
  -d '{"value": "us-east-1"}'
```

### 3. Execute Commands Using Variables
```bash
curl -X POST http://localhost:8080/api/v1/execute \
  -H 'Content-Type: application/json' \
  -d '{"command": "print \"Deploying to @environment in @region\""}'
```

### 4. Start Background Job
```bash
curl -X POST http://localhost:8080/api/v1/execute \
  -H 'Content-Type: application/json' \
  -d '{"command": "osexec --background \"sleep 60\""}'
```

### 5. List Running Jobs
```bash
curl http://localhost:8080/api/v1/jobs
```

### 6. Get System Info
```bash
curl http://localhost:8080/api/v1/info
```

## Integration Examples

### Python Client

```python
import requests
import json

BASE_URL = "http://localhost:8080/api/v1"

# Execute command
response = requests.post(
    f"{BASE_URL}/execute",
    json={"command": "date"}
)
result = response.json()
print(f"Output: {result['output']}")
print(f"Duration: {result['duration_ms']}ms")

# List jobs
response = requests.get(f"{BASE_URL}/jobs")
jobs = response.json()
print(f"Running jobs: {jobs['count']}")

# Set variable
requests.put(
    f"{BASE_URL}/variables/myvar",
    json={"value": "hello"}
)

# Execute using variable
response = requests.post(
    f"{BASE_URL}/execute",
    json={"command": "print @myvar"}
)
print(response.json()['output'])
```

### JavaScript/Node.js Client

```javascript
const axios = require('axios');

const BASE_URL = 'http://localhost:8080/api/v1';

// Execute command
async function executeCommand(cmd) {
  const response = await axios.post(`${BASE_URL}/execute`, {
    command: cmd
  });
  return response.data;
}

// List jobs
async function listJobs() {
  const response = await axios.get(`${BASE_URL}/jobs`);
  return response.data;
}

// Example usage
(async () => {
  // Execute command
  const result = await executeCommand('date');
  console.log('Output:', result.output);
  console.log('Duration:', result.duration_ms, 'ms');

  // List jobs
  const jobs = await listJobs();
  console.log('Jobs:', jobs.count);
})();
```

### Bash Script

```bash
#!/bin/bash

API="http://localhost:8080/api/v1"

# Execute command
execute() {
  curl -s -X POST "$API/execute" \
    -H 'Content-Type: application/json' \
    -d "{\"command\": \"$1\"}" | jq -r '.output'
}

# List jobs
list_jobs() {
  curl -s "$API/jobs" | jq '.jobs'
}

# Example usage
execute "print 'Hello from bash'"
list_jobs
```

## Error Handling

All API endpoints return appropriate HTTP status codes:

- `200 OK` - Request successful
- `400 Bad Request` - Invalid request body or parameters
- `404 Not Found` - Resource not found (job, variable)
- `500 Internal Server Error` - Server error

**Error Response Format:**
```json
{
  "error": "Error message description"
}
```

## Security Considerations

### Production Deployment

1. **Add Authentication**: Implement API key or JWT authentication
2. **Rate Limiting**: Prevent abuse with rate limits
3. **HTTPS Only**: Use TLS/SSL in production
4. **Input Validation**: Validate all command inputs
5. **Command Whitelist**: Restrict allowed commands

### Example Authentication Middleware

```go
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        apiKey := r.Header.Get("X-API-Key")
        if apiKey != os.Getenv("API_KEY") {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Apply to router
apiRouter.Use(authMiddleware)
```

## Monitoring & Logging

### Enable Command Logging

```go
customizer := func(exec *consolekit.CommandExecutor) error {
    exec.LogManager.Enable()
    exec.LogManager.SetLogFile("/var/log/api-commands.log")
    // ... rest of setup
}
```

### Prometheus Metrics

Add metrics endpoint:

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

router.Handle("/metrics", promhttp.Handler())
```

## Troubleshooting

### Port Already in Use

Change port in `main.go`:
```go
httpHandler := consolekit.NewHTTPHandler(executor, ":8081", "admin", "secret123")
```

### CORS Issues

Add CORS headers in middleware (already included in example).

### JSON Parse Errors

Ensure `Content-Type: application/json` header is set in requests.

## See Also

- [Multi-Transport Example](../multi_transport/) - Combined transports
- [SSH Server Example](../ssh_server/) - SSH access
- [Main Documentation](../../CLAUDE.md) - Full ConsoleKit documentation
