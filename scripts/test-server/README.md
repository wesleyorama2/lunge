# High-Performance Test Server

Simple Go HTTP server optimized for load testing.

## Quick Start

```bash
# Build and run
cd scripts/test-server
docker-compose up --build

# Or build and run manually
docker build -t test-server .
docker run -p 80:80 test-server
```

## Endpoints

- `GET /status/200` - Returns "OK" (mimics httpbin)
- `GET /health` - Health check

## Performance

This server is optimized for high throughput:
- Minimal processing overhead
- Uses all available CPU cores
- Optimized timeouts and connection handling
- Should easily handle 1000+ RPS on modern hardware
