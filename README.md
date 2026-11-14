# Tavrocred — Personal P2P Loan Tracking Platform

Tavrocred is a personal peer-to-peer loan recordkeeping and communication platform. It enables individuals to record loans they make to people they know, track manual repayments made outside the application (via UPI, bank transfer, cash, etc.), and send consent-based reminders.

> **Note**: Tavrocred is a recordkeeping and communication product, not a payment processor, lender, credit marketplace, collection agency, or escrow service.

## Architecture

- **Backend**: Go modular monolith (`services/api`) with Chi router, PostgreSQL (`pgx/v5`), sqlc query generation, and Goose migrations.
- **Frontend**: Next.js responsive web application (`apps/web`).
- **Contracts**: OpenAPI 3.1 specification (`packages/contracts/openapi.yaml`).
- **Local Dev**: Docker Compose for PostgreSQL 16, Mailpit, API, Worker, and Web.

## Milestone 1 Implemented Features

- **Authentication & Security**:
  - Secure registration and login with Argon2id password hashing.
  - Short-lived JWT access tokens and cryptographically random rotating refresh tokens with SHA-256 hash storage.
  - Refresh token rotation and instant revocation on logout.
  - Authenticated user profile endpoint (`/api/v1/auth/me`).
- **Borrower / Peer Management**:
  - Full CRUD operations (`POST`, `GET`, `PUT`, `DELETE`) for borrowers under `/api/v1/peers`.
  - Strict ownership isolation at both query predicate and service layers (preventing cross-tenant data leaks).
  - Search and pagination support (`page`, `page_size`, `q`).
- **Infrastructure & Operational Excellence**:
  - Health (`/healthz`) and Readiness (`/readyz`) probes.
  - Request ID tracing (`X-Request-ID`), structured logging with `log/slog`, panic recovery, and configurable request timeouts.
  - Docker Compose orchestrating PostgreSQL, Mailpit, API, background worker, and web frontend.
  - Complete OpenAPI 3.1 contract.

## Quick Start

### Prerequisites
- Go 1.22+
- Docker & Docker Compose
- Make

### Environment Setup

Copy example environment variables:
```bash
cp .env.example .env
```

### Running with Docker Compose

```bash
docker compose up --build
```

Access:
- API: `http://localhost:8080`
- API Health: `http://localhost:8080/healthz`
- API Readiness: `http://localhost:8080/readyz`
- Mailpit Web UI: `http://localhost:8025`
- Web Frontend: `http://localhost:3000`

### Running Locally

```bash
# Start database and Mailpit
docker compose up -d postgres mailpit

# Run tests
make test

# Build binaries
make build
```

