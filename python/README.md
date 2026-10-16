# ROGELE Analytics

A Python service that provides analytics and reporting for the Rogele
learning platform.

The service consumes the ROGELE Go API over HTTP. It never connects directly
to the database. Its role is to compute the analyses that are easier in Python
than in Go: trend analysis, cohort retention, distributions, forecasting, and
eventually machine learning.

## Requirements

- Python 3.12 or later
- [uv](https://github.com/astral-sh/uv) for dependency management
- A running ROGELE Go API
- A valid service token (see below)

## Installation

    uv sync

## Configuration

Copy `.env.example` to `.env` and fill in the values:

    ANALYTICS_APP_NAME=ROGELE Analytics
    ANALYTICS_DEBUG=true
    ANALYTICS_PORT=9000

    ROGELEDB_API_URL=http://localhost:8080
    ROGELEDB_SERVICE_TOKEN=

    HTTP_TIMEOUT=30

`ROGELEDB_SERVICE_TOKEN` is a JWT issued by the ROGELE Go API. The analytics
service uses it to authenticate as an admin when calling protected endpoints.

To obtain a token for local development, log in as the admin user and copy the
access token:

    $admin = Invoke-RestMethod -Method POST http://localhost:8080/api/v1/auth/login `
      -ContentType 'application/json' `
      -Body '{"email":"admin@rogele.dev","password":"supersecret123"}'
    $admin.access_token

Paste the value into `.env`. Admin access tokens are short-lived (15 minutes
by default). A dedicated long-lived service-token endpoint is planned.

## Running

Start the service:

    uv run uvicorn rogeledb_analytics.main:app --reload --port 9000

The service listens on port 9000. Health check:

    curl http://localhost:9000/health

Readiness check (verifies the service can reach the Go API):

    curl http://localhost:9000/health/upstream

## Running with Docker

    docker compose up --build

The container reaches the Go API on the host through
`host.docker.internal`. Set `ROGELEDB_API_URL` accordingly in `.env`:

    ROGELEDB_API_URL=http://host.docker.internal:8080

## Project Layout

    src/rogeledb_analytics/
      __init__.py
      config.py                Environment-backed settings
      go_client.py             Async HTTP client for the Go API
      main.py                  FastAPI application
      routes/                  Endpoint modules
        health.py

## Endpoints

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | /health | Liveness |
| GET | /health/upstream | Readiness (verifies Go API is reachable) |

Additional endpoints are added as features are implemented.

## Development

Format and lint:

    uv run ruff format .
    uv run ruff check --fix .

Type-check:

    uv run mypy src

## Design Notes

- The service is read-only against the Go API. It never writes.
- All requests carry a bearer token from the configured service account.
- Computation is done in pandas and numpy. The output is always JSON that
  the frontend can render directly — no chart rendering on the server.
- No business logic lives here. Business rules stay in the Go API, which
  remains the single source of truth.