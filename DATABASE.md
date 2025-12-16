# Database Configuration

## Connection String Format

```
postgresql://username:password@host:port/database?sslmode=disable
```

## Production Examples

### PostgreSQL (Recommended for Production)

```env
DATABASE_URL=postgresql://springstreet_user:password@db-host:5432/springstreet?sslmode=disable
```

### Docker Compose (Internal Network)

```env
DATABASE_URL=postgresql://springstreet:password@db:5432/springstreet
```

### Remote PostgreSQL

```env
DATABASE_URL=postgresql://user:pass@example.com:5432/dbname?sslmode=require
```

## Environment Variables

Set these in your production environment:

```env
DATABASE_URL=postgresql://user:password@host:5432/database?sslmode=disable
SECRET_KEY=your-strong-secret-key-here
PORT=8000
HOST=0.0.0.0
DEBUG=false
```

## Security Notes

- Use strong passwords for production
- Enable SSL (`sslmode=require`) for remote connections
- Never commit `.env` files to version control
- Use secrets management (Docker secrets, Kubernetes secrets, etc.)

