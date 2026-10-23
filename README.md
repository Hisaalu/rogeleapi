# ROGELE API

ROGELE is a learning management platform for schools. This repository contains
the backend HTTP API that serves the web application, the Android application,
and internal analytics services.

The API is written in Go and uses PostgreSQL as its primary datastore. It
exposes a REST interface for authentication, academic content (lessons,
homework, quizzes), subscriptions, payments, notifications, and administrative
tooling.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Requirements](#requirements)
- [Installation](#installation)
- [Configuration](#configuration)
- [Running the API](#running-the-api)
- [Database Migrations](#database-migrations)
- [Project Layout](#project-layout)
- [API Reference](#api-reference)
- [Authentication](#authentication)
- [Authorization Model](#authorization-model)
- [File Uploads](#file-uploads)
- [Payments](#payments)
- [Notifications](#notifications)
- [Email](#email)
- [Error Format](#error-format)
- [Pagination and Filtering](#pagination-and-filtering)
- [Development Workflow](#development-workflow)
- [Deployment](#deployment)
- [License](#license)

## Overview

The ROGELE API is a single Go binary that provides:

- JWT authentication with access and refresh tokens.
- Role-based access control for admin, teacher, learner, and external users.
- Lesson management with class scoping, publishing, approval, bookmarks, and
  materials.
- Homework assignment, submission, and grading.
- Quizzes with question banks, timed attempts, automatic scoring, and
  resumable sessions.
- Subscription plans, free trials, and payment processing through MTN Mobile
  Money and Airtel Money.
- Notifications with per-user read state, broadcasts, and event hooks.
- Transactional and campaign email delivery.
- File uploads to Cloudflare R2 through presigned URLs.
- A generic, YAML-driven CRUD engine for low-complexity resources.

## Architecture

The service is organized into four layers:

1. Handlers receive HTTP requests and delegate to services.
2. Services implement business rules and orchestrate repositories.
3. Repositories isolate SQL behind interfaces.
4. Storage adapters (R2, MTN, Airtel, Resend) talk to external systems.

Long-running work (email delivery, expiry reminders) is handled by background
workers inside the same process. Work is queued in PostgreSQL so the HTTP layer
never blocks on network I/O.

The Python analytics service reads from the same database and provides
aggregated insights (at-risk learners, enrollment trends, score distribution)
to the web client.

## Requirements

- Go 1.22 or later
- PostgreSQL 14 or later
- golang-migrate CLI

## Installation

    git clone https://github.com/Hisaalu/rogeleapi.git
    cd rogeleapi
    go mod tidy

Install the migration CLI:

    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

## Configuration

Configuration is provided through a `.env` file at the project root. Values in
`.env` override `configs/config.yaml`. Environment variables take precedence
over both.

    # Application
    ROGELEDB_APP_NAME=rogele
    ROGELEDB_APP_ENV=development
    ROGELEDB_APP_PORT=8080

    # Database
    ROGELEDB_DB_DRIVER=postgres
    ROGELEDB_DB_HOST=127.0.0.1
    ROGELEDB_DB_PORT=5432
    ROGELEDB_DB_USER=postgres
    ROGELEDB_DB_PASSWORD=
    ROGELEDB_DB_NAME=rogeledb
    ROGELEDB_DB_SSLMODE=disable
    ROGELEDB_DB_MAX_OPEN_CONNS=25
    ROGELEDB_DB_MAX_IDLE_CONNS=5
    ROGELEDB_DB_CONN_MAX_LIFETIME=1h

    # JWT
    ROGELEDB_JWT_SECRET=
    ROGELEDB_JWT_ACCESS_TTL=15m
    ROGELEDB_JWT_REFRESH_TTL=168h

    # MTN Mobile Money
    ROGELEDB_MTN_MOMO_BASE_URL=
    ROGELEDB_MTN_MOMO_SUBSCRIPTION_KEY=
    ROGELEDB_MTN_MOMO_API_USER=
    ROGELEDB_MTN_MOMO_API_KEY=
    ROGELEDB_MTN_MOMO_TARGET_ENV=sandbox
    ROGELEDB_MTN_MOMO_CALLBACK_URL=
    ROGELEDB_MTN_MOMO_CURRENCY=EUR

    # Airtel Money
    ROGELEDB_AIRTEL_BASE_URL=
    ROGELEDB_AIRTEL_CLIENT_ID=
    ROGELEDB_AIRTEL_CLIENT_SECRET=
    ROGELEDB_AIRTEL_CALLBACK_URL=

    # Cloudflare R2
    ROGELEDB_R2_ACCOUNT_ID=
    ROGELEDB_R2_ACCESS_KEY_ID=
    ROGELEDB_R2_SECRET_ACCESS_KEY=
    ROGELEDB_R2_BUCKET=
    ROGELEDB_R2_PUBLIC_URL=
    ROGELEDB_R2_ENDPOINT=

    # Email
    ROGELEDB_RESEND_API_KEY=
    ROGELEDB_EMAIL_FROM=
    ROGELEDB_EMAIL_REPLY_TO=
    ROGELEDB_WEB_BASE_URL=

The `.env` file must not be committed.

## Running the API

    go run ./cmd/api

Health check:

    curl http://localhost:8080/health

Build a binary:

    go build -o bin/rogele ./cmd/api

## Database Migrations

Migrations live in `migrations/` and are applied with the `migrate` CLI. The
password in the connection URL must be URL-encoded (`/` becomes `%2F`).

    migrate -path ./migrations \
      -database "postgres://user:pass@127.0.0.1:5432/rogeledb?sslmode=disable" up

Common commands:

    migrate -path ./migrations -database "$URL" up          # apply all
    migrate -path ./migrations -database "$URL" down 1      # roll back one
    migrate -path ./migrations -database "$URL" version     # show version
    migrate create -ext sql -dir ./migrations -seq name     # new migration pair

For databases provisioned before migrations were introduced, mark the current
version without running migrations:

    migrate -path ./migrations -database "$URL" force 1

## Project Layout

    cmd/api/                Application entry point
    configs/                YAML configuration and CRUD resource definitions
    internal/
      auth/                 JWT signing and password hashing
      config/               Configuration loader
      crud/                 Generic YAML-driven CRUD engine
      db/                   PostgreSQL connection pool
      dberr/                Mapping of PostgreSQL errors to domain errors
      domain/               Domain models and sentinel errors
      dto/                  Request and response shapes
      handler/              HTTP handlers
      httputil/             JSON response helpers
      logger/               Structured logging
      mailer/               Email delivery, branded shell, queue worker
      payments/             MTN and Airtel adapters
      repository/           Repository interfaces
        postgres/           PostgreSQL implementations
      scheduler/            Background schedulers
      server/               Router and middleware
      service/              Business logic
      storage/              Cloudflare R2 client and presigned URLs
    migrations/             golang-migrate SQL files

## API Reference

All endpoints are prefixed with `/api/v1`. Unless noted otherwise, requests
must include `Authorization: Bearer <access_token>`.

### Authentication

| Method | Path | Description |
| ------ | ---- | ----------- |
| POST | /auth/register | Register a new user |
| POST | /auth/login | Authenticate and receive tokens |
| POST | /auth/refresh | Exchange a refresh token |
| POST | /auth/forgot-password | Request a password reset link |
| POST | /auth/reset-password | Complete a password reset |
| GET | /auth/me | Return the authenticated user |

### Lessons

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | /lessons | List lessons visible to the caller |
| POST | /lessons | Create a lesson |
| GET | /lessons/bookmarks | List bookmarked lessons |
| GET | /lessons/{id} | Retrieve a lesson |
| PATCH | /lessons/{id} | Update a lesson |
| DELETE | /lessons/{id} | Delete a lesson |
| POST | /lessons/{id}/publish | Publish or unpublish |
| POST | /lessons/{id}/approve | Approve (admin) |
| POST | /lessons/{id}/bookmark | Toggle bookmark |
| GET | /lessons/{id}/materials | List materials |
| POST | /lessons/{id}/materials | Register a material |
| DELETE | /lessons/{id}/materials/{materialId} | Delete a material |

### Homework

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | /homework | List homework |
| POST | /homework | Create homework |
| GET | /homework/my-submissions | List the caller's submissions |
| GET | /homework/{id} | Retrieve homework |
| PATCH | /homework/{id} | Update homework |
| DELETE | /homework/{id} | Delete homework |
| POST | /homework/{id}/submit | Submit homework |
| GET | /homework/{id}/submissions | List submissions |
| POST | /homework/submissions/{submissionId}/grade | Grade a submission |
| GET | /homework/{id}/attachments | List attachments |
| POST | /homework/{id}/attachments | Register an attachment |
| DELETE | /homework/{id}/attachments/{attachmentId} | Delete an attachment |
| GET | /homework/submissions/{submissionId}/files | List submission files |
| POST | /homework/submissions/{submissionId}/files | Register a submission file |
| DELETE | /homework/submissions/{submissionId}/files/{fileId} | Delete a submission file |

### Quizzes

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | /quizzes | List quizzes |
| POST | /quizzes | Create a quiz |
| GET | /quizzes/my-attempts | List the caller's attempts |
| GET | /quizzes/{id} | Retrieve a quiz |
| PATCH | /quizzes/{id} | Update a quiz |
| DELETE | /quizzes/{id} | Delete a quiz |
| POST | /quizzes/{id}/approve | Approve (admin) |
| GET | /quizzes/{id}/questions | List questions |
| POST | /quizzes/{id}/questions | Add a question |
| DELETE | /quizzes/{id}/questions/{questionId} | Delete a question |
| POST | /quizzes/{id}/attempts | Start or resume an attempt |
| GET | /quizzes/{id}/attempts | List attempts (teacher or admin) |
| DELETE | /quizzes/{id}/attempts | Delete all attempts |
| GET | /quizzes/attempts/{attemptId} | Retrieve an attempt |
| POST | /quizzes/attempts/{attemptId}/submit | Submit an attempt |

### Subscriptions and Trials

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | /subscriptions/me | Current active subscription |
| GET | /subscriptions | List the caller's subscriptions |
| GET | /subscriptions/all | List all (admin) |
| POST | /subscriptions/{id}/cancel | Cancel a subscription |
| GET | /trials/me | Return the caller's trial |
| POST | /trials | Start a free trial |

### Payments

| Method | Path | Description |
| ------ | ---- | ----------- |
| POST | /payments/initiate | Initiate a payment |
| GET | /payments/me | List the caller's payments |
| GET | /payments/history | List the caller's payment history |
| GET | /payments/all | List all (admin) |
| GET | /payments/{id}/status | Retrieve payment status |
| POST | /payments/preview | Calculate prorated upgrade balance |

### Notifications

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | /notifications/me | List notifications visible to the caller |
| GET | /notifications/me/unread-count | Count unread notifications |
| POST | /notifications/{id}/read | Mark one notification read |
| POST | /notifications/read-all | Mark every visible notification read |

Notifications are addressed to a single user (`user_id` set) or broadcast to
everyone (`user_id` null). Read state is tracked per user in
`notification_reads`, so marking a broadcast read for one user never affects
another.

Notifications are emitted automatically on:

- New homework assigned to a class.
- A quiz is published to a class.
- A lesson is published to a class.
- A payment succeeds or fails.
- A subscription is close to expiring.
- A free trial is close to ending.

### Webhooks

| Method | Path | Description |
| ------ | ---- | ----------- |
| POST | /webhooks/mtn | MTN Mobile Money callback |
| POST | /webhooks/airtel | Airtel Money callback |

### Uploads

| Method | Path | Description |
| ------ | ---- | ----------- |
| POST | /uploads/presign | Generate a presigned upload URL |

### Admin

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | /admin/stats | Platform-wide statistics |
| GET | /admin/activity | Recent activity feed |
| GET | /admin/recent-signups | Recent user registrations |
| GET | /admin/recent-payments | Recent payments |
| GET | /admin/users/{id}/attempts | Quiz attempts for a user |
| GET | /admin/users/{id}/submissions | Homework submissions for a user |
| POST | /admin/users/{id}/force-reset | Send a password reset link |
| POST | /admin/users/{id}/set-password | Directly set a user's password |
| POST | /admin/notifications/broadcast | Send a notification to every user |
| GET | /admin/notifications | List sent broadcasts |

### Email Campaigns (Admin)

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | /admin/email-campaigns | List campaigns |
| POST | /admin/email-campaigns | Create a draft campaign |
| GET | /admin/email-campaigns/{id} | Retrieve a campaign |
| PATCH | /admin/email-campaigns/{id} | Update a draft campaign |
| DELETE | /admin/email-campaigns/{id} | Delete a draft campaign |
| POST | /admin/email-campaigns/{id}/preview | Preview a rendered campaign |
| POST | /admin/email-campaigns/{id}/send | Queue the campaign |
| GET | /admin/email-campaigns/{id}/logs | Per-recipient delivery logs |

### Generic CRUD Resources

Exposed by the YAML-driven CRUD engine in `configs/resources.yaml`:

- `/settings`
- `/subscription-plans`
- `/classes`
- `/subjects`
- `/email-templates`
- `/users`

Each supports `GET /`, `GET /{id}`, `POST /`, `PATCH /{id}`, and `DELETE /{id}`,
subject to role restrictions declared in the resource definition.

## Authentication

Login and registration return two JWTs:

- Access token, short-lived (default 15 minutes).
- Refresh token, long-lived (default 7 days).

Send the access token in the `Authorization` header. When it expires:

    POST /api/v1/auth/refresh
    { "refresh_token": "..." }

Refresh tokens are accepted only at the refresh endpoint. Using a refresh
token elsewhere returns 401 `WRONG_TOKEN_TYPE`.

## Authorization Model

Three layers:

1. **Authentication.** All endpoints except the public auth routes and payment
   webhooks require a valid access token.
2. **Role-based access.** Each endpoint declares which roles may invoke it.
3. **Resource ownership.** Teachers may only modify their own lessons,
   homework, and quizzes. Admins may modify anything. Learners may only read
   resources matching their class.

A service token can be generated for internal use (analytics, public stats).
This token has `role: admin` and is used by the web client to fetch public data
without requiring a user session.

## File Uploads

Files go directly from client to Cloudflare R2 using presigned URLs. The API
never receives file bytes.

1. `POST /api/v1/uploads/presign` with `file_name`, `content_type`, and `size`.
2. The response contains `upload_url`, `file_key`, and `public_url`. Valid 15
   minutes.
3. Upload with an HTTP `PUT` to `upload_url`, using the same `Content-Type`.
4. Register the file with the resource endpoint, passing `public_url` as the
   file path.

Limits:

- Maximum file size: 5 MB.
- Allowed content types: PDF, common Office formats, plain text, CSV, JPEG,
  PNG, WebP, GIF, MP4, QuickTime, WebM, MP3, M4A.

## Payments

    POST /api/v1/payments/initiate
    {
      "plan_id": 1,
      "method": "mtn",
      "phone_number": "250788000000"
    }

The API calls the provider, records a payment with status `pending`, and
returns a provider reference. The provider confirms asynchronously via the
webhook endpoints.

On success, the API activates or extends the subscription, writes to
`payment_history`, sends a payment confirmation email, and emits a
notification. Failed payments are recorded with status `failed` and also emit
a notification.

The `/payments/preview` endpoint calculates the prorated balance for plan
upgrades, so the web client can display the exact amount before payment.

## Notifications

Read state is stored in `notification_reads`, keyed by
`(notification_id, user_id)`. A broadcast has one shared row and one read-state
row per user who has seen it.

The frontend polls `/notifications/me/unread-count` every 30 seconds. Clicking
a notification marks it read and follows its link if present.

## Email

Email is delivered asynchronously. Outgoing messages are written to
`email_queue` and processed by a background worker. Every attempt is recorded
in `email_logs`.

Transactional emails:

- Password reset
- Payment confirmation
- Subscription expiring soon
- Trial ending soon

Administrators may create campaigns through the email campaign endpoints.
Campaigns resolve recipients by role, by explicit user IDs, or by custom email
addresses. Delivery status is recorded per recipient. A background reconciler
marks campaigns as `completed` or `failed`.

Templates live in `email_templates` and may be edited at runtime. Four are
seeded on startup. Every email is wrapped in a branded shell that reads the
school name, logo, and colors from `settings`. Changes take effect within 60
seconds.

## Error Format

    {
      "error": {
        "code": "INVALID_INPUT",
        "message": "invalid input: password must be at least 8 characters"
      }
    }

| Code | HTTP Status | Meaning |
| ---- | ----------- | ------- |
| INVALID_INPUT | 400 | Validation failed |
| UNAUTHORIZED | 401 | Missing or invalid credentials |
| INVALID_TOKEN | 401 | Token malformed or expired |
| WRONG_TOKEN_TYPE | 401 | Refresh token where access token expected |
| FORBIDDEN | 403 | Authenticated but not permitted |
| NOT_FOUND | 404 | Resource does not exist |
| CONFLICT | 409 | Uniqueness or state conflict |
| INTERNAL_ERROR | 500 | Unhandled server error |

## Pagination and Filtering

List endpoints accept:

- `page` (default 1)
- `per_page` (default 20, maximum 100)
- `q` for free-text search where supported
- resource-specific filters
- `sort` for comma-separated sort keys, prefixed with `-` for descending

    {
      "data": [ ... ],
      "meta": {
        "page": 1,
        "per_page": 20,
        "total": 143,
        "last_page": 8
      }
    }

## Development Workflow

    make run              # Run the API
    make build            # Build a binary
    make tidy             # Tidy dependencies
    make fmt              # Format the codebase
    make vet              # Static analysis
    make test             # Run tests
    make migrate-up       # Apply migrations
    make migrate-down     # Roll back one migration
    make migrate-create name=add_some_table

Before committing:

    go build ./...
    go vet ./...
    gofmt -w .

The API uses `log/slog`. Development logs are text; production logs are JSON.
Every request is tagged with a request ID returned in `X-Request-ID`.

## Deployment

    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/rogele ./cmd/api

Run migrations before starting the new binary:

    migrate -path ./migrations -database "$DATABASE_URL" up

Configuration comes from environment variables. `/health` is available for
liveness and readiness checks. Graceful shutdown drains in-flight requests and
stops background workers on `SIGINT` or `SIGTERM`.

## License

This project is proprietary. All rights reserved.