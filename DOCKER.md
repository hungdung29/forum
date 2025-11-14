# Docker Deployment Guide

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Build and start the container
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the container
docker-compose down

# Rebuild after code changes
docker-compose up -d --build
```

The forum will be available at http://localhost:8080

### Using Docker CLI

```bash
# Build the image
docker build -t forum-app .

# Run the container
docker run -d \
  --name forum \
  -p 8080:8080 \
  -v forum-data:/app/server/database \
  -e ENV=production \
  forum-app

# View logs
docker logs -f forum

# Stop and remove
docker stop forum && docker rm forum
```

## Features

### Multi-Stage Build
- **Stage 1 (Builder)**: Compiles Go binary with optimizations
- **Stage 2 (Runtime)**: Minimal Alpine image (~20MB vs ~400MB)
- Benefits: Faster deployment, smaller attack surface, reduced bandwidth

### Security
- Non-root user (`appuser`) for running the application
- Static binary with stripped symbols
- Minimal runtime dependencies
- Read-only filesystem support (except `/app/server/database`)

### Health Checks
Built-in health check endpoint at `/health` monitors:
- Database connectivity
- Disk space (warns at 85%, fails at 95%)
- Memory usage
- Response time

Docker automatically checks health every 30s:
```bash
docker ps  # Shows "healthy" status
```

### Automatic Migrations
On startup, the container automatically:
1. Initializes the `schema_migrations` table
2. Applies pending migrations in order
3. Seeds demo data (if migration 002 is pending)

### Environment Variables

Configure via `docker-compose.yml` or `-e` flags:

```bash
# Server
PORT=8080                    # HTTP port
ENV=production              # Environment (production/development)

# Database
DB_PATH=server/database/database.db
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# Timeouts
READ_TIMEOUT=15s
WRITE_TIMEOUT=15s
IDLE_TIMEOUT=60s

# Application
BASE_PATH=/app/
APP_VERSION=1.0.0

# Cache
CACHE_TEMPLATE_TTL=1h
CACHE_SESSION_TTL=10m
CACHE_POST_TTL=5m
```

### Persistent Data

The database is stored in a Docker volume:
```bash
# List volumes
docker volume ls

# Inspect volume
docker volume inspect forum_forum-data

# Backup database
docker run --rm \
  -v forum_forum-data:/data \
  -v $(pwd):/backup \
  alpine cp /data/database.db /backup/backup.db

# Restore database
docker run --rm \
  -v forum_forum-data:/data \
  -v $(pwd):/backup \
  alpine cp /backup/backup.db /data/database.db
```

## Troubleshooting

### View Container Logs
```bash
docker-compose logs -f forum
```

### Execute Commands in Container
```bash
# Get a shell
docker-compose exec forum sh

# Check health manually
docker-compose exec forum wget -O- http://localhost:8080/health
```

### Rebuild from Scratch
```bash
# Remove everything and rebuild
docker-compose down -v
docker-compose up -d --build
```

### Check Health Status
```bash
curl http://localhost:8080/health | jq
```

Expected output:
```json
{
  "status": "healthy",
  "timestamp": "2025-11-13T16:50:00Z",
  "version": "1.0.0",
  "uptime": "5m 30s",
  "checks": {
    "database": {
      "status": "pass",
      "message": "Connected",
      "time": "2ms"
    },
    "disk": {
      "status": "pass",
      "message": "50.25 GB available (15.3% used)"
    },
    "memory": {
      "status": "pass",
      "message": "Alloc: 3.45 MB, Sys: 12.34 MB"
    }
  }
}
```

## Production Deployment

### Recommended Setup

1. **Use Docker Compose with Production Config**:
   ```yaml
   environment:
     - ENV=production
     - APP_VERSION=1.0.0
   restart: always
   ```

2. **Set Up Reverse Proxy** (Nginx/Traefik):
   ```nginx
   upstream forum {
       server localhost:8080;
   }

   server {
       listen 80;
       server_name forum.example.com;

       location / {
           proxy_pass http://forum;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
       }

       location /health {
           access_log off;
           proxy_pass http://forum/health;
       }
   }
   ```

3. **Monitor Health Checks**:
   - Integrate with Prometheus/Grafana
   - Set up alerts for unhealthy status
   - Monitor disk space warnings

4. **Backup Strategy**:
   ```bash
   # Automated daily backups
   0 2 * * * docker run --rm -v forum_forum-data:/data -v /backups:/backup alpine sh -c "cp /data/database.db /backup/forum-$(date +\%Y\%m\%d).db"
   ```

5. **Log Aggregation**:
   - Use Docker logging drivers (e.g., `json-file`, `syslog`)
   - Forward logs to centralized system (ELK, Loki)

## Performance Optimization

### Image Size Comparison
- Before (single-stage): ~400 MB
- After (multi-stage): ~20 MB
- Savings: **95% smaller**

### Build Time
- Docker layer caching optimizes rebuilds
- Only `go mod download` if dependencies change
- Binary compilation cached unless code changes

### Runtime Performance
- Template caching: 10x faster rendering
- Connection pooling: Efficient database usage
- Rate limiting: Prevents resource exhaustion
- Graceful shutdown: Zero dropped requests

## Demo Credentials

The seed migration creates test users with password `password123`:
- alice@example.com / alice
- bob@example.com / bob
- charlie@example.com / charlie
- diana@example.com / diana
- eve@example.com / eve

**⚠️ Change these in production!**
