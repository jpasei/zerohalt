# Zerohalt

> **⚠️ ALPHA SOFTWARE**: This project is in early development. APIs and behavior may change without notice.

A lightweight process manager for containers that provides graceful shutdown coordination with zero-downtime deployments.

## Features

- Graceful shutdown with connection draining
- Built-in health check server with multiple modes
- Application health verification with startup timeout
- Active connection monitoring via `/proc/net/tcp`
- Prometheus metrics export for observability
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
export ZEROHALT_APP_PORT=8080                           # Primary port to monitor for connections
export ZEROHALT_APP_HEALTH_URL=http://localhost:8080/health  # App health endpoint (for app-dependent mode)
export ZEROHALT_APP_STARTUP_TIMEOUT=30s                 # Max time to wait for app to become healthy

# Health check settings
export ZEROHALT_HEALTH_PORT=8888                        # Health check server port
export ZEROHALT_HEALTH_PATH=/health                     # Health check endpoint path
export ZEROHALT_HEALTH_MODE=standalone                  # Mode: standalone, app-dependent
export ZEROHALT_HEALTH_PROBE_INTERVAL=1s                # Interval for app health checks

# Shutdown settings
export ZEROHALT_DRAIN_TIMEOUT=60s                       # Max time to wait for connections to drain
export ZEROHALT_DRAIN_STEADY_STATE_WAIT=5s              # Wait time at zero connections before proceeding
export ZEROHALT_SHUTDOWN_TIMEOUT=30s                    # Max time to wait for app to exit
export ZEROHALT_SIGNAL_TO_APP=SIGTERM                   # Signal to send to app on shutdown (empty = forward received signal)

# Signal forwarding
export ZEROHALT_PASSTHROUGH_SIGNALS=SIGHUP,SIGUSR1      # Signals to forward to app
export ZEROHALT_SHUTDOWN_SIGNALS=SIGTERM,SIGINT         # Signals that trigger shutdown

# Metrics (optional)
export ZEROHALT_METRICS_ENABLED=true                    # Enable Prometheus metrics
export ZEROHALT_METRICS_PORT=8888                       # Metrics server port (can share with health)
export ZEROHALT_METRICS_PATH=/metrics                   # Metrics endpoint path

# Logging
export ZEROHALT_LOG_LEVEL=info                          # Log level: debug, info, warn, error
```

**Note**: Additional ports monitoring and force-kill configuration are planned but not yet implemented via environment variables.

## Health Check Modes

Zerohalt's health endpoint (`ZEROHALT_HEALTH_PORT`) reflects the lifecycle state of your container with the following states:

| State | HTTP Status | Description |
|-------|-------------|-------------|
| **Starting (0)** | 503 | Application process is launching |
| **Healthy (1)** | 200 | Application is running and healthy |
| **Unhealthy (2)** | 503 | Application health check is failing (app-dependent mode only) |
| **Draining (3)** | 503 | Graceful shutdown in progress, draining connections |
| **Terminating (4)** | 503 | Final shutdown phase |

### Standalone Mode (Default)

```bash
export ZEROHALT_HEALTH_MODE=standalone
```

Zerohalt sets the health state to **Healthy** immediately after starting your application process. Use this mode when your application doesn't expose its own health endpoint.

### App-Dependent Mode

```bash
export ZEROHALT_HEALTH_MODE=app-dependent
export ZEROHALT_APP_HEALTH_URL=http://localhost:8080/health
export ZEROHALT_APP_STARTUP_TIMEOUT=30s
export ZEROHALT_HEALTH_PROBE_INTERVAL=1s
```

Zerohalt actively monitors your application's health endpoint:
- Waits for app to become healthy before marking container ready
- Continuously probes app health and can transition **Unhealthy → Healthy** automatically
- If app fails to become healthy within `STARTUP_TIMEOUT`, Zerohalt logs a warning but **continues running** (does not crash)
- Container remains operational even if app is unhealthy, allowing investigation and recovery

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

1. **Startup**:
   - Starts health server in **Starting** state
   - Launches your application process
   - Waits for app health (app-dependent mode) or marks **Healthy** immediately (standalone mode)

2. **Running**:
   - Monitors active connections on configured ports via `/proc/net/tcp`
   - Forwards pass-through signals to application
   - Reaps zombie processes (proper PID 1 behavior)
   - Exports Prometheus metrics (if enabled)

3. **Shutdown**:
   - Receives shutdown signal (SIGTERM/SIGINT)
   - Marks health state as **Draining** (returns 503)
   - Waits for connections to drain (respects `DRAIN_TIMEOUT`)
   - Sends configured signal to application
   - Waits for graceful app exit (respects `SHUTDOWN_TIMEOUT`)
   - Force kills if timeout exceeded and `FORCE_KILL=true`

## Prometheus Metrics

When metrics are enabled (`ZEROHALT_METRICS_ENABLED=true`), Zerohalt exposes Prometheus metrics at the configured endpoint:

```
# Health and state metrics
zerohalt_state                    # Current health state (0-4, see states above)
zerohalt_health_app               # Application health state (0-4, matches state enum)
zerohalt_uptime_seconds           # Zerohalt uptime
zerohalt_app_uptime_seconds       # Managed application uptime

# Connection metrics
zerohalt_active_connections       # Current active connections
zerohalt_drain_phase_active       # 1 if draining, 0 otherwise
zerohalt_drain_duration_seconds   # Time spent draining connections

# Health endpoint metrics
zerohalt_health_requests_total    # Total health check requests
zerohalt_health_request_duration_ms  # Health check latency

# Signal metrics
zerohalt_signals_received_total{signal}   # Signals received by Zerohalt
zerohalt_signals_forwarded_total{signal}  # Signals forwarded to app
```

**State Values**: Both `zerohalt_state` and `zerohalt_health_app` use the same enum:
- 0 = Starting
- 1 = Healthy
- 2 = Unhealthy
- 3 = Draining
- 4 = Terminating

This consistent scale allows tracking state transitions and correlating health changes across time series.

## Roadmap

### Current Status (v0.1.0 - Alpha)

Zerohalt has matured significantly with core features implemented and 100% test coverage across critical packages.

**✅ Completed:**
- Core process management (PID 1, signal handling, zombie reaping)
- Health check HTTP server with full lifecycle states (Starting, Healthy, Unhealthy, Draining, Terminating)
- App-dependent health mode with continuous health verification
- Application startup timeout with automatic recovery support
- Connection monitoring via `/proc/net/tcp` (IPv4 and IPv6)
- Graceful shutdown coordination with connection draining
- Steady-state wait during drain phase
- Signal pass-through and forwarding
- Prometheus metrics export with full state tracking
- Environment variable configuration
- Multi-architecture builds (amd64, arm64)
- 100% test coverage for core packages (health, process, shutdown, main)

**🚧 In Progress:**
- Multiple ports monitoring via environment variables
- Hybrid and command-based health check modes
- Force-kill configuration via environment variables
- CLI flags support (currently env vars only)
- Integration tests with real containers
- End-to-end testing in Kubernetes

**📋 Planned:**
- CI/CD pipeline with automated releases
- Performance benchmarks and optimization
- Security audit and hardening
- Configuration file support (YAML/TOML)
- Comprehensive deployment examples and production guides

### Development Phases

**Phase 1: Enhanced Health Modes** (Next)
- Implement hybrid health mode (Zerohalt + App health combined)
- Add command-based health checks
- Support for custom health check commands

**Phase 2: Production Hardening**
- Integration and E2E test suites
- GitHub Actions CI/CD pipeline
- Performance optimization (binary size, memory usage)
- Security hardening and audit
- Production deployment documentation

**Phase 3: Advanced Features**
- Advanced shutdown strategies
- Configuration file support (YAML/TOML)
- CLI flags and command-line interface improvements
- Additional metrics and observability features

### Contributing

Contributions are welcome! This project is in early stages and needs help with:
- Testing and bug reports
- Documentation improvements
- Feature implementation
- Real-world usage feedback

## License

Apache License 2.0 - Copyright 2025 JPA Solution Experts, Inc.

See [LICENSE](LICENSE) file for details.
