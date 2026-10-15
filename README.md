# ROGELE API

ROGELE is a learning management platform for schools. This repository contains
the backend HTTP API that serves the web application, the Android application,
and internal analytics services.

The API is written in Go and uses PostgreSQL as its primary datastore. It exposes
a REST interface for authentication, academic content (lessons, homework,
quizzes), subscriptions, payments, notifications, and administrative tooling.

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
- [Testing](#testing)
- [Deployment](#deployment)
- [License](#license)

## Overview

The ROGELE API is a single Go binary that provides:

- JWT-based authentication with access and refresh tokens.
- Role-based access control for four user roles: admin, teacher, learner, and
  external user.
- Lesson management with class scoping, publishing, approval, bookmarks, and
  attached materials.
- Homework assignment, submission, and grading.
- Quizzes with question banks, timed attempts, automatic scoring, and resumable
  sessions.
- Subscription plans, free trials, subscriptions, and payment processing through
  MTN Mobile Money and Airtel Money.
- Notifications with per-user read state, broadcasts, and event hooks across
  every major subsystem.
- Email delivery for password resets, payment confirmations, subscription
  reminders, trial reminders, and administrator-driven campaigns.
- File uploads to Cloudflare R2 through presigned URLs.
- A generic, YAML-driven CRUD engine for low-complexity resources.
- A branded email shell driven by runtime settings.

## Architecture

The service is organized into four layers:

1. Handlers receive HTTP requests, decode input, and delegate to services.
2. Services implement business rules and orchestrate repositories.
3. Repositories isolate SQL and persistence concerns behind interfaces.
4. Storage adapters (R2, MTN, Airtel, Resend) talk to external systems.

Long-running work (email delivery, expiry reminders) is handled by background
workers that run inside the same process. Work is queued in PostgreSQL so that
the HTTP layer never blocks on network I/O to external providers.

The application is designed so that all clients — web, mobile, analytics —
communicate exclusively with the API. No client connects directly to the
database.

## Requirements

- Go 1.22 or later
- PostgreSQL 14 or later
- golang-migrate CLI
- Docker (optional, for local PostgreSQL)

## Installation

Clone the repository:

    git clone https://github.com/Hisaalu/rogeleapi.git
    cd rogeleapi

Install dependencies:

    go mod tidy

Install the golang-migrate CLI:

    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

## Configuration

Configuration is provided through a `.env` file at the project root. The
application loads `.env` on startup and overrides values from
`configs/config.yaml`. Environment variables take precedence over YAML.

Create a `.env` file with the following keys:

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

The `.env` file must not be committed to version control. It is listed in
`.gitignore`.

## Running the API

Start the server:

    go run ./cmd/api

The server listens on the port specified by `ROGELEDB_APP_PORT`. Health check:

    curl http://localhost:8080/health

Build a binary:

    go build -o bin/rogele ./cmd/api

Run the binary:

    ./bin/rogele

## Database Migrations

Migrations live in `migrations/` and are applied with the `migrate` CLI.

Define a reusable connection string in your shell, or use the `Makefile`
targets. The password in the connection URL must be URL-encoded (for example,
`/` becomes `%2F`).

    migrate -path ./migrations \
      -database "postgres://user:pass@127.0.0.1:5432/rogeledb?sslmode=disable" up

Common commands:

    # Apply all pending migrations
    migrate -path ./migrations -database "$URL" up

    # Roll back the most recent migration
    migrate -path ./migrations -database "$URL" down 1

    # Show the current version
    migrate -path ./migrations -database "$URL" version

    # Create a new migration pair
    migrate create -ext sql -dir ./migrations -seq add_some_table

For databases that were provisioned before migrations were introduced, mark the
current version without running migrations:

    migrate -path ./migrations -database "$URL" force 1

The migration set includes:

- `000001_init` — baseline schema (users, classes, subjects, lessons, homework,
  quizzes, subscriptions, payments, notifications, email).
- `000002_email` — the `email_queue` table for async delivery.
- `000003_email_template_unique` — unique constraint on `email_templates.name`.
- `000004_expiry_reminder` — `subscriptions.expiry_reminder_sent` flag.
- `000005_updated_at` — `updated_at` columns on `classes` and `subjects`.
- `000006_unique_classes_subjects` — uniqueness on class names and subject
  names per class.
- `000007_timestamptz_quizzes` — timestamp columns on quizzes and attempts
  converted to `timestamptz`.
- `000008_notifications_user` — per-user notifications (`notifications.user_id`).
- `000009_notification_reads` — per-user read state for broadcast
  notifications.

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
      mailer/               Email delivery, branded shell, queue worker, templates
      payments/             Provider abstraction with MTN and Airtel adapters
      repository/           Repository interfaces
        postgres/           PostgreSQL implementations
      scheduler/            Background schedulers (expiry reminders)
      server/               Router and middleware
      service/              Business logic
      storage/              Cloudflare R2 client and presigned URLs
    migrations/             golang-migrate SQL files

## API Reference

All endpoints are prefixed with `/api/v1`. Unless noted otherwise, requests
must include a valid `Authorization` header:

    Authorization: Bearer <access_token>

### Authentication

| Method | Path | Description |
| ------ | ---- | ----------- |
| POST | /auth/register | Register a new user |
| POST | /auth/login | Authenticate and receive tokens |
| POST | /auth/refresh | Exchange a refresh token for a new access token |
| POST | /auth/forgot-password | Request a password reset link |
| POST | /auth/reset-password | Complete a password reset |
| GET | /auth/me | Return the authenticated user |

### Lessons

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | /lessons | List lessons visible to the caller |
| POST | /lessons | Create a lesson (teacher or admin) |
| GET | /lessons/bookmarks | List lessons bookmarked by the caller |
| GET | /lessons/{id} | Retrieve a lesson |
| PATCH | /lessons/{id} | Update a lesson |
| DELETE | /lessons/{id} | Delete a lesson |
| POST | /lessons/{id}/publish | Publish or unpublish a lesson |
| POST | /lessons/{id}/approve | Approve or unapprove a lesson (admin) |
| POST | /lessons/{id}/bookmark | Toggle a bookmark |
| GET | /lessons/{id}/materials | List lesson materials |
| POST | /lessons/{id}/materials | Register a material |
| DELETE | /lessons/{id}/materials/{materialId} | Delete a material |

### Homework

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | /homework | List homework visible to the caller |
| POST | /homework | Create homework (teacher or admin) |
| GET | /homework/my-submissions | List the caller's submissions |
| GET | /homework/{id} | Retrieve homework |
| PATCH | /homework/{id} | Update homework |
| DELETE | /homework/{id} | Delete homework |
| POST | /homework/{id}/submit | Submit homework (learner) |
| GET | /homework/{id}/submissions | List submissions for homework |
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
| GET | /quizzes | List quizzes visible to the caller |
| POST | /quizzes | Create a quiz (teacher or admin) |
| GET | /quizzes/my-attempts | List the caller's quiz attempts |
| GET | /quizzes/{id} | Retrieve a quiz |
| PATCH | /quizzes/{id} | Update a quiz |
| DELETE | /quizzes/{id} | Delete a quiz |
| POST | /quizzes/{id}/approve | Approve or unapprove a quiz (admin) |
| GET | /quizzes/{id}/questions | List questions for a quiz |
| POST | /quizzes/{id}/questions | Add a question |
| DELETE | /quizzes/{id}/questions/{questionId} | Delete a question |
| POST | /quizzes/{id}/attempts | Start or resume an attempt |
| GET | /quizzes/{id}/attempts | List all attempts for a quiz (teacher or admin) |
| DELETE | /quizzes/{id}/attempts | Delete all attempts for a quiz (teacher or admin) |
| GET | /quizzes/attempts/{attemptId} | Retrieve an attempt |
| POST | /quizzes/attempts/{attemptId}/submit | Submit an attempt |

### Subscriptions and Trials

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | /subscriptions/me | Return the caller's active subscription |
| GET | /subscriptions | List the caller's subscriptions |
| GET | /subscriptions/all | List all subscriptions (admin) |
| POST | /subscriptions/{id}/cancel | Cancel a subscription |
| GET | /trials/me | Return the caller's trial |
| POST | /trials | Start a free trial (external users) |

### Payments

| Method | Path | Description |
| ------ | ---- | ----------- |
| POST | /payments/initiate | Initiate a payment |
| GET | /payments/me | List the caller's payments |
| GET | /payments/history | List the caller's payment history |
| GET | /payments/all | List all payments (admin) |
| GET | /payments/{id}/status | Retrieve payment status |

### Notifications

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | /notifications/me | List notifications visible to the caller |
| GET | /notifications/me/unread-count | Count unread notifications |
| POST | /notifications/{id}/read | Mark one notification read |
| POST | /notifications/read-all | Mark every visible notification read |

Notifications may be addressed to a single user (`user_id` set) or broadcast
to everyone (`user_id` null). Read state is tracked per user in the
`notification_reads` table, so marking a broadcast as read for one user never
affects another user.

Notifications are emitted automatically on the following events:

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
| POST | /admin/users/{id}/force-reset | Send a password reset link to a user |
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
| POST | /admin/email-campaigns/{id}/send | Queue the campaign for delivery |
| GET | /admin/email-campaigns/{id}/logs | List per-recipient delivery logs |

### Generic CRUD Resources

The following resources are exposed by the YAML-driven CRUD engine, defined in
`configs/resources.yaml`:

- `/settings`
- `/subscription-plans`
- `/classes`
- `/subjects`
- `/email-templates`
- `/users`

Each resource supports `GET /`, `GET /{id}`, `POST /`, `PATCH /{id}`, and
`DELETE /{id}`, subject to role restrictions declared in the resource
definition.

## Authentication

The API issues two JWTs on login and registration:

- Access token with a short lifetime (default 15 minutes).
- Refresh token with a longer lifetime (default 7 days).

Send the access token on every request in the `Authorization` header:

    Authorization: Bearer <access_token>

When the access token expires, exchange the refresh token:

    POST /api/v1/auth/refresh
    {
      "refresh_token": "..."
    }

Refresh tokens are accepted only at the refresh endpoint. Passing a refresh
token to any other endpoint results in a 401 response with the error code
`WRONG_TOKEN_TYPE`.

## Authorization Model

The API enforces three layers of authorization:

1. Authentication. All endpoints except `/auth/register`, `/auth/login`,
   `/auth/refresh`, `/auth/forgot-password`, `/auth/reset-password`, and the
   payment webhooks require a valid access token.

2. Role-based access. Each endpoint declares which roles may invoke it.
   Attempts from other roles return 403.

3. Resource ownership. Teachers may only modify lessons, homework, and quizzes
   they created. Admins may modify any resource. Learners may only read
   resources matching their class.

## File Uploads

Files are uploaded directly from the client to Cloudflare R2 using presigned
URLs. The API never receives file bytes.

Upload flow:

1. `POST /api/v1/uploads/presign` with a JSON body containing `file_name`,
   `content_type`, and `size`.
2. The response contains `upload_url`, `file_key`, and `public_url`. The
   presigned URL is valid for 15 minutes.
3. Upload the file with an HTTP `PUT` to `upload_url`, using the same
   `Content-Type` that was included in the presign request.
4. Register the file with the relevant resource endpoint, passing `public_url`
   as the file path.

Limits:

- Maximum file size: 5 MB.
- Allowed content types: PDF, common Office formats, plain text, CSV, JPEG,
  PNG, WebP, GIF, MP4, QuickTime, WebM, MP3, M4A.

## Payments

Payments are initiated with a plan ID, a payment method, and a phone number:

    POST /api/v1/payments/initiate
    {
      "plan_id": 1,
      "method": "mtn",
      "phone_number": "250788000000"
    }

The API calls the provider, records a payment with status `pending`, and
returns a provider reference. The provider confirms the outcome asynchronously
via the webhook endpoints.

When a payment is confirmed as successful, the API activates or extends the
user's subscription, writes an entry to `payment_history`, sends a payment
confirmation email, and emits a notification. Failed payments are recorded with
status `failed` and also emit a notification.

Airtel support is planned and the provider interface is in place; credentials
are required to enable it.

## Notifications

Every notification belongs to either one user (`user_id` set) or every user
(`user_id` null, a broadcast).

Read state is stored in `notification_reads` — a table keyed by
`(notification_id, user_id)`. A broadcast notification therefore has one shared
row and one read-state row per user who has seen it.

The frontend polls `/notifications/me/unread-count` every 30 seconds to keep
the badge current. Clicking a notification in the UI marks it read and, if it
has a link, navigates to that destination.

## Email

Email is delivered asynchronously. Outgoing messages are written to the
`email_queue` table and processed by a background worker that runs inside the
API process. Every delivery attempt is recorded in `email_logs`.

The following transactional emails are supported:

- Password reset
- Payment confirmation
- Subscription expiring soon
- Trial ending soon

Administrators may create campaigns through the email campaign endpoints.
Campaigns resolve recipients by role, by explicit user IDs, or by a list of
custom email addresses. Each recipient receives a rendered message; delivery
status is recorded per recipient. A background reconciler marks campaigns as
`completed` or `failed` once all queued jobs have been processed.

Templates are stored in `email_templates` and may be edited at runtime. Four
templates are seeded on startup.

Every email is wrapped in a branded shell. The shell reads the school name,
logo, primary color, accent color, and website URL from the `settings` table.
Changes to branding take effect on subsequent emails within 60 seconds.

## Error Format

Errors are returned with a consistent JSON envelope:

    {
      "error": {
        "code": "INVALID_INPUT",
        "message": "invalid input: password must be at least 8 characters"
      }
    }

Common error codes:

| Code | HTTP Status | Meaning |
| ---- | ----------- | ------- |
| INVALID_INPUT | 400 | Validation failed |
| UNAUTHORIZED | 401 | Missing or invalid credentials |
| INVALID_TOKEN | 401 | Token is malformed or expired |
| WRONG_TOKEN_TYPE | 401 | Refresh token used where access token expected |
| FORBIDDEN | 403 | Authenticated but not permitted |
| NOT_FOUND | 404 | Resource does not exist |
| CONFLICT | 409 | Uniqueness violation or state conflict |
| INTERNAL_ERROR | 500 | Unhandled server error |

## Pagination and Filtering

List endpoints accept the following query parameters:

- `page` (default 1)
- `per_page` (default 20, maximum 100)
- `q` for free-text search where supported
- resource-specific filters documented in the endpoint description
- `sort` for comma-separated sort keys, prefixed with `-` for descending order

Responses use a common envelope:

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

Common commands:

    # Run the API
    make run

    # Build a binary
    make build

    # Tidy dependencies
    make tidy

    # Format the codebase
    make fmt

    # Run static analysis
    make vet

    # Run tests
    make test

    # Apply migrations
    make migrate-up

    # Roll back one migration
    make migrate-down

    # Create a new migration
    make migrate-create name=add_some_table

Before committing, run:

    go build ./...
    go vet ./...
    gofmt -w .

The API uses `log/slog` for structured logging. In development, logs are
emitted as text. In production, logs are emitted as JSON. Every request is
tagged with a request ID that is returned in the `X-Request-ID` response
header.

## Testing

Run the test suite:

    go test ./... -race -cover

Tests are colocated with the code they exercise and named `*_test.go`.

## Deployment

The service is distributed as a single static binary. Build it with:

    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/rogele ./cmd/api

Run migrations before starting the new binary:

    migrate -path ./migrations -database "$DATABASE_URL" up

The binary reads configuration from environment variables and exits on startup
if any required value is missing. A `/health` endpoint is available for
liveness and readiness checks.

Graceful shutdown is implemented. The server drains in-flight requests and
stops background workers when it receives `SIGINT` or `SIGTERM`.

## License

This project is proprietary. All rights reserved.