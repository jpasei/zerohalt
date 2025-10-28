# Zerohalt

> **⚠️ ALPHA SOFTWARE**: This project is in early development. APIs and behavior may change without notice.

A lightweight process manager for containers that provides graceful shutdown coordination with zero-downtime deployments.

## Features

- Graceful shutdown with connection draining
- Built-in health check server
- Active connection monitoring via `/proc/net/tcp`
- Signal pass-through for application reload
- Zombie process reaping (proper PID 1 behavior)
- Zero runtime dependencies
- Supports linux/amd64 and linux/arm64

## Building

### Build for all platforms
```bash
./build.sh
```

The binaries will be created in `build/zerohalt`.

### Run tests
```bash
./test.sh
```

## Deploying to Applications

### Basic Dockerfile Usage

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o zerohalt ./cmd/zerohalt

FROM alpine:latest
COPY --from=builder /build/zerohalt /usr/local/bin/zerohalt
COPY your-app /usr/local/bin/your-app

# Use Zerohalt as the entrypoint
ENTRYPOINT ["/usr/local/bin/zerohalt"]
CMD ["your-app", "--port", "8080"]
```

### Configuration via Environment Variables

Configure Zerohalt using environment variables:

```bash
# Application settings
export ZEROHALT_APP_PORT=8080                  # Port(s) to monitor for connections

# Health check settings
export ZEROHALT_HEALTH_PORT=8888               # Health check server port
export ZEROHALT_HEALTH_PATH=/health            # Health check endpoint path

# Shutdown settings
export ZEROHALT_DRAIN_TIMEOUT=60s              # Max time to wait for connections to drain
export ZEROHALT_SHUTDOWN_TIMEOUT=30s           # Max time to wait for app to exit
export ZEROHALT_SIGNAL_TO_APP=SIGTERM          # Signal to send to app on shutdown

# Signal forwarding
export ZEROHALT_PASSTHROUGH_SIGNALS=SIGHUP,SIGUSR1  # Signals to forward to app
export ZEROHALT_SHUTDOWN_SIGNALS=SIGTERM,SIGINT     # Signals that trigger shutdown

# Logging
export ZEROHALT_LOG_LEVEL=info                 # Log level: debug, info, warn, error
```

**Additional ports**: To monitor multiple ports for connections, set additional ports via the config (environment variable support coming soon).

## Health Check Modes

Zerohalt's health endpoint (`ZEROHALT_HEALTH_PORT`) reflects the lifecycle state of your container:
- **Starting**: Returns 503 while the application process is launching
- **Healthy**: Returns 200 once the application is running
- **Draining**: Returns 503 during graceful shutdown (from signal received until application exits)

### Current Implementation

Zerohalt sets the health state to healthy immediately after starting your application process. The health endpoint tracks Zerohalt's lifecycle state, not your application's actual readiness.

**Note**: Health check modes and custom health verification are planned features but not yet implemented.

### Kubernetes Deployment Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: myapp
        image: myapp:latest
        command: ["/usr/local/bin/zerohalt"]
        args: ["myapp", "--port", "8080"]
        env:
        - name: ZEROHALT_HEALTH_PORT
          value: "8888"
        - name: ZEROHALT_APP_PORT
          value: "8080"
        - name: ZEROHALT_DRAIN_TIMEOUT
          value: "60s"
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 8888
          name: health
        livenessProbe:
          httpGet:
            path: /health
            port: 8888
        readinessProbe:
          httpGet:
            path: /health
            port: 8888
```

## How It Works

Zerohalt runs as PID 1 in your container and manages your application:

1. **Startup**: Starts your application and health server
2. **Running**: Monitors connections, forwards signals, reaps zombie processes
3. **Shutdown**: On SIGTERM → marks unhealthy → drains connections → signals app → waits for graceful exit

## Roadmap

### Current Status (v0.1.0 - Alpha)

Zerohalt is in active development. Core functionality is implemented but several features are incomplete.

**✅ Completed:**
- Core process management (PID 1, signal handling, zombie reaping)
- Basic health check HTTP server with lifecycle states
- Connection monitoring via `/proc/net/tcp` (IPv4 and IPv6)
- Graceful shutdown coordination with connection draining
- Signal pass-through and forwarding
- Environment variable configuration
- Multi-architecture builds (amd64, arm64)

**🚧 In Progress:**
- Health check modes (standalone, app-dependent, hybrid, command)
- Application health verification and startup timeout
- Additional ports monitoring via environment variables
- CLI flags support

**📋 Planned:**
- Complete test coverage (unit, integration, e2e)
- CI/CD pipeline with automated releases
- Performance benchmarks and optimization
- Security audit and hardening
- Comprehensive examples and documentation
- Production deployment guides
- Optional Prometheus metrics export

### Development Phases

**Phase 1: Complete Health Check System** (Next)
- Implement app-dependent health mode with startup verification
- Add command-based health checks
- Support for custom health check commands
- Environment variables for all health settings

**Phase 2: Testing & Quality**
- Achieve 80%+ code coverage
- Integration tests with real containers
- End-to-end testing in Kubernetes
- GitHub Actions CI/CD pipeline

**Phase 3: Production Readiness**
- Performance optimization (binary size <10MB, memory <5MB)
- Security hardening and audit
- Error handling improvements
- Production deployment documentation

**Phase 4: Enhanced Features**
- Multiple port monitoring via environment variables
- Advanced shutdown strategies
- Metrics and observability
- Configuration file support (YAML/TOML)

### Contributing

Contributions are welcome! This project is in early stages and needs help with:
- Testing and bug reports
- Documentation improvements
- Feature implementation
- Real-world usage feedback

## License

Apache License 2.0 - Copyright 2025 JPA Solution Experts, Inc.

See [LICENSE](LICENSE) file for details.
