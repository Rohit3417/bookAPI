# bookAPI
# bookAPI

A RESTful Book Management API built with Go, featuring JWT authentication, PostgreSQL persistence (via Supabase), and Redis-backed token blacklisting for logout.

## Features

- **CRUD operations** for books (create, list, get by ID, delete)
- **JWT-based authentication** with signup/login/logout
- **Token blacklisting** on logout using Redis, so logged-out tokens are rejected even before expiry
- **Password hashing** with bcrypt
- Built on [chi](https://github.com/go-chi/chi) router with middleware for request logging, panic recovery, request IDs, and CORS

## Tech Stack

| Layer      | Technology                          |
|------------|--------------------------------------|
| Language   | Go                                   |
| Router     | chi v5                              |
| Database   | PostgreSQL (Supabase)               |
| DB Driver  | pgx/v5 (pgxpool)                    |
| Cache      | Redis (Upstash) via go-redis/v9     |
| Auth       | JWT (golang-jwt/jwt/v5) + bcrypt    |
| Config     | godotenv                            |

## Project Structure

```
bookAPI/
├── cmd/
│   └── main.go              # entrypoint, routing, middleware
├── internal/
│   ├── auth/
│   │   └── auth.go          # JWT generation & validation
│   ├── cache/
│   │   └── redisConn.go     # Redis client + token blacklisting
│   ├── database/
│   │   └── db.go            # Postgres connection pool
│   └── handler/
│       ├── AuthHandler.go   # signup/login/logout handlers
│       └── BookHandler.go   # book CRUD handlers
├── .env.example
├── go.mod
└── README.md
```

## Getting Started

### Prerequisites

- Go 1.21+
- A Supabase project (PostgreSQL)
- An Upstash Redis instance (or any Redis instance with TLS support)

### Environment Variables

Copy `.env.example` to `.env` and fill in your own values:

```
# Server
PORT=8080

# Postgres (Supabase - use the Session Pooler connection details)
DB_HOST=aws-0-<region>.pooler.supabase.com
DB_PORT=5432
DB_USER=postgres.<project-ref>
DB_PASSWORD=your-db-password
DB_NAME=postgres

# Redis (Upstash)
REDIS_ADDR=your-endpoint.upstash.io:6379
REDIS_PASSWORD=your-upstash-password

# Auth
JWT_SECRET=your-long-random-secret
```

### Database Schema

Run the following in the Supabase SQL Editor:

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);

CREATE TABLE books (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    author TEXT NOT NULL
);
```

### Install & Run

```bash
go mod tidy
go run ./cmd/main.go
```

Server starts on `http://localhost:8080` (or the port set in `PORT`).

## API Endpoints

### Auth

| Method | Endpoint         | Description                          | Auth Required |
|--------|------------------|---------------------------------------|----------------|
| POST   | `/auth/signup`   | Register a new user                   | No             |
| POST   | `/auth/login`    | Log in, returns a JWT                 | No             |
| POST   | `/auth/logout`   | Blacklist the current token           | Yes (Bearer)   |

**Signup / Login request body:**
```json
{
  "username": "rohit",
  "password": "yourpassword"
}
```

**Login response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### Books

| Method | Endpoint       | Description         |
|--------|----------------|----------------------|
| GET    | `/books`       | List all books       |
| GET    | `/books/{id}`  | Get a book by ID     |
| POST   | `/books`       | Create a new book     |
| DELETE | `/books/{id}`  | Delete a book by ID  |

**Create book request body:**
```json
{
  "title": "Dune",
  "author": "Frank Herbert"
}
```

## Example Usage (curl)

```bash
# Signup
curl -X POST http://localhost:8080/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"username":"rohit","password":"testpass123"}'

# Login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"rohit","password":"testpass123"}'

# Create a book
curl -X POST http://localhost:8080/books \
  -H "Content-Type: application/json" \
  -d '{"title":"Dune","author":"Frank Herbert"}'

# List books
curl http://localhost:8080/books
```

## Error Response Format

All errors follow a consistent shape:

```json
{
  "error": "invalid credentials",
  "code": 401,
  "timestamp": "2026-07-23T10:15:00Z"
}
```

## Notes on Supabase / Upstash Connectivity

- Use the **Session Pooler** connection string from Supabase (not the direct `db.xxxx.supabase.co` host) if your network doesn't support IPv6 — the pooler is IPv4-compatible.
- Upstash Redis requires TLS; the Redis client must be configured with `TLSConfig` to connect successfully.

## Roadmap

- [ ] Re-enable JWT middleware on `/books` routes
- [ ] Add `/health` endpoint for deployment health checks
- [ ] Graceful shutdown on SIGTERM
- [ ] Deploy to Railway

## License

MIT
