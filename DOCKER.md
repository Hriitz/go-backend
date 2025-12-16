# Docker Setup Guide

## Quick Start

### Build and Run with Docker

```bash
# Build the image
docker build -t springstreet-api:latest .

# Run the container
docker run -p 8000:8000 \
  -e DATABASE_URL=postgresql://springstreet_user:springstreet_password@host.docker.internal:5432/springstreet \
  -e SECRET_KEY=your-secret-key \
  -e PORT=8000 \
  springstreet-api:latest
```

### Using Docker Compose

```bash
# Start all services (API + PostgreSQL + Redis + pgAdmin)
docker-compose up -d

# View logs
docker-compose logs -f backend-go

# Stop all services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

## Step-by-Step Dockerization

### Step 1: Build the Docker Image

```bash
cd backend-go
docker build -t springstreet-api:latest .
```

**What happens:**
- Uses multi-stage build (builder + runtime)
- Installs Go 1.22 and dependencies
- Generates Goa code from design files
- Builds optimized binary (`springstreet-api`)
- Creates minimal Alpine-based runtime image (~20-30MB)

### Step 2: Run the Container

#### Option A: Standalone (SQLite)
```bash
docker run -d \
  --name springstreet-api \
  -p 8000:8000 \
  -e DATABASE_URL=sqlite:///./spring_street.db \
  -e SECRET_KEY=your-secret-key-here \
  -e PORT=8000 \
  springstreet-api:latest
```

#### Option B: With Docker Compose (PostgreSQL)
```bash
docker-compose up -d
```

This starts:
- **backend-go**: API server on port 8000
- **db**: PostgreSQL database on port 5432
- **redis**: Redis cache on port 6379
- **pgadmin**: Database admin UI on port 5050

### Step 3: Verify the Container

```bash
# Check if container is running
docker ps

# Check logs
docker logs springstreet-api

# Test health endpoint
curl http://localhost:8000/health
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `sqlite:///./spring_street.db` | Database connection string |
| `SECRET_KEY` | `your-secret-key-change-in-production` | JWT secret key |
| `PORT` | `8000` | Server port |
| `HOST` | `0.0.0.0` | Server host |
| `DEBUG` | `false` | Debug mode |

## Database Options

### SQLite (Default - Development)
```bash
DATABASE_URL=sqlite:///./spring_street.db
```

### PostgreSQL (Production)
```bash
# For docker-compose (internal network)
DATABASE_URL=postgresql://springstreet:password@db:5432/springstreet

# For standalone container connecting to host PostgreSQL
DATABASE_URL=postgresql://springstreet_user:password@host.docker.internal:5432/springstreet
```

## Production Deployment

1. **Set strong SECRET_KEY**: Use a secure random string
   ```bash
   openssl rand -base64 32
   ```

2. **Use PostgreSQL**: SQLite is not recommended for production

3. **Configure CORS**: Update allowed origins in the code

4. **Use HTTPS**: Deploy behind a reverse proxy (nginx/traefik)

5. **Set resource limits**: Configure CPU/memory limits in docker-compose

6. **Use Docker Secrets or Environment Files**:
   ```bash
   docker run -d \
     --env-file .env.production \
     springstreet-api:latest
   ```

## Troubleshooting

### Container Exits Immediately
```bash
# Check logs
docker logs springstreet-api

# Common issues:
# - Database connection failed
# - Port already in use
# - Missing environment variables
```

### Database Connection Issues
```bash
# For PostgreSQL in docker-compose
docker-compose exec db psql -U springstreet -d springstreet

# Check if database is accessible
docker-compose exec backend-go ping db

# Verify DATABASE_URL format
docker-compose exec backend-go env | grep DATABASE_URL
```

### Build Fails
```bash
# Clean build
docker build --no-cache -t springstreet-api:latest .

# Check Go version compatibility
docker run --rm golang:1.22 go version

# Verify all source files are present
```

## Health Checks

The container includes automatic health checks:
```bash
# Manual health check
curl http://localhost:8000/health

# Docker health status
docker inspect --format='{{.State.Health.Status}}' springstreet-api
```

## Monitoring

### Logs
```bash
# Follow logs
docker logs -f springstreet-api

# Last 100 lines
docker logs --tail 100 springstreet-api
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: Build and Push Docker Image

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build Docker image
        run: docker build -t springstreet-api:${{ github.sha }} .
      - name: Push to registry
        run: |
          docker tag springstreet-api:${{ github.sha }} registry.example.com/springstreet-api:latest
          docker push registry.example.com/springstreet-api:latest
```

