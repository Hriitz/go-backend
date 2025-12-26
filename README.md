# Spring Street Backend API (Go/Goa)

Production-ready Go backend API for Spring Street - Global investing platform for Indian investors.

## 🚀 Features

- **Goa Framework** - Design-first API development
- **GORM** - Database ORM (SQLite/PostgreSQL)
- **JWT Authentication** - Secure token-based auth
- **Bcrypt Password Hashing** - Industry-standard security
- **Clean Architecture** - Domain-driven design
- **Standard Go Layout** - Follows Go best practices

## 📁 Project Structure

```
backend-go/
├── api/design/            # Goa API design files
├── cmd/                   # Application entry points
│   ├── api/              # Main API server
│   └── create_admin/      # Admin user creation tool
├── internal/             # Private application code
│   ├── config/           # Configuration management
│   ├── database/         # Database connection & migration
│   ├── domain/           # Domain models
│   ├── models/           # Data models
│   ├── services/         # Business logic layer
│   └── util/             # Utility functions
├── pkg/                  # Public packages
│   └── errors/           # Error definitions
├── scripts/              # Build scripts
│   └── generate.sh       # Goa code generation
├── Dockerfile            # Production Docker image
├── docker-compose.yml    # Production deployment stack
└── Makefile             # Build automation
```

## 🚀 Quick Start (Production)

### Prerequisites

- Docker and Docker Compose installed
- PostgreSQL database (or use docker-compose)

### Deploy with Docker Compose (Recommended)

```bash
# Clone repository
git clone <repository-url>
cd backend-go

# Start all services (API + PostgreSQL + Redis)
docker-compose up -d

# View logs
docker-compose logs -f backend-go

# Check health
curl http://localhost:8000/health
```

### Deploy with Docker

```bash
# Build the image
docker build -t springstreet-api:latest .

# Run container
docker run -d \
  --name springstreet-api \
  -p 8000:8000 \
  -e DATABASE_URL=postgresql://user:password@host:5432/database \
  -e SECRET_KEY=your-secret-key \
  springstreet-api:latest
```

See [DOCKER.md](DOCKER.md) for detailed deployment instructions.

## 📚 Documentation

- [DOCKER.md](DOCKER.md) - Docker deployment guide
- [DATABASE.md](DATABASE.md) - Database configuration

## 🔧 Build Commands

```bash
# Generate Goa code
make gen

# Build binary
make build

# Run tests
make test
```

## 📡 API Endpoints

- Health: `GET /health`
- Auth: `POST /api/v1/auth/login`
- Investment: `POST /api/v1/investment/`
- OTP: `POST /api/v1/otp/send`

## 🔐 Security

- ✅ Bcrypt password hashing
- ✅ JWT token authentication
- ✅ Role-based access control
- ✅ CORS configuration
- ✅ Input validation

