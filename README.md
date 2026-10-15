# ROGELE WEB

The web client for ROGELE, a learning management platform for schools.

This application is the browser interface for three audiences:

- Administrators, who manage users, classes, subjects, subscription plans, and
  platform settings.
- Teachers, who create lessons, assign homework, build quizzes, and grade
  submissions.
- Learners and external users, who browse lessons, submit homework, take
  quizzes, and manage their subscriptions.

The web client communicates exclusively with the ROGELE API. It never connects
directly to a database or any other backend service.

## Table of Contents

- [Overview](#overview)
- [Tech Stack](#tech-stack)
- [Requirements](#requirements)
- [Installation](#installation)
- [Configuration](#configuration)
- [Running Locally](#running-locally)
- [Project Structure](#project-structure)
- [Authentication](#authentication)
- [Branding and Theming](#branding-and-theming)
- [Notifications](#notifications)
- [API Communication](#api-communication)
- [Roles and Routes](#roles-and-routes)
- [Development Workflow](#development-workflow)
- [Building for Production](#building-for-production)
- [Deployment](#deployment)
- [License](#license)

## Overview

The ROGELE Web app is a Next.js application built with the App Router. It
provides three distinct experiences behind a single authentication system:

- `/` and `/dashboard` — the learner and teacher experience
- `/admin` and sub-pages — the administration panel
- `/login` and related pages — public authentication flows

The layout is responsive, supports light and dark themes, and pulls its
branding — school name, tagline, logo, and colors — from the API at runtime.

## Tech Stack

- Next.js 14 (App Router)
- React 18
- TypeScript (strict mode)
- Tailwind CSS
- Lucide icons
- React Hook Form with Zod validation
- TanStack Query for server state
- shadcn-style UI primitives built on Radix

## Requirements

- Node.js 18.17 or later
- pnpm 9 or later (npm and yarn work too)

## Installation

Clone the repository:

    git clone https://github.com/Hisaalu/rogele-web.git
    cd rogele-web

Install dependencies:

    pnpm install

## Configuration

Configuration lives in `.env.local` at the project root. It is not committed.

    # Go API base URL (server-side, no trailing slash)
    ROGELEDB_API_URL=http://localhost:8080

    # Same URL, exposed to the browser for reference
    NEXT_PUBLIC_API_URL=http://localhost:8080

    # Session secret used to sign auth cookies
    SESSION_SECRET=change-me-to-a-long-random-string-at-least-32-chars

    # Public URL of this app
    NEXT_PUBLIC_APP_URL=http://localhost:3000

`ROGELEDB_API_URL` is used by server components and the internal proxy route.
`SESSION_SECRET` signs the auth cookies that hold JWTs. It must be at least 32
characters and must differ between environments.

## Running Locally

Start the development server:

    pnpm dev

The app is available at http://localhost:3000.

The ROGELE API must be running and reachable at `ROGELEDB_API_URL` for
authentication and data to work.

## Project Structure

    src/
      app/
        (auth)/                Public auth pages
          login/
          register/
          forgot-password/
          reset-password/
        (app)/                 Learner and teacher experience
          dashboard/
          lessons/
          homework/
          quizzes/
          subscription/
          profile/
          notifications/
        (admin)/               Admin panel
          admin/
            page.tsx           Overview
            users/
            classes/
            subjects/
            plans/
            subscriptions/
            payments/
            notifications/
            email-campaigns/
            settings/
        api/
          auth/                Login, logout, refresh routes
          proxy/               Catch-all proxy to the Go API
        layout.tsx
        globals.css
      components/
        brand/                 Logo, branding provider
        layout/                Sidebar, header, user menu
        notifications/         Bell, list
        ui/                    Buttons, dialogs, inputs, tables, etc.
      lib/
        api-client.ts          Typed fetch wrapper
        auth.ts                Cookie-based session helpers
        server-api.ts          Server-side API client
        branding.ts            Fetches branding from the API
        *.server.ts            Server-only fetchers per resource
        *.client.ts            Client mutators per resource
      types/                   TypeScript models matching the API DTOs

## Authentication

The web app uses httpOnly cookies for tokens. The browser never sees the JWT
directly.

Flow:

1. The login form posts credentials to `/api/auth/login`, a Next.js route
   handler that proxies to `POST /api/v1/auth/login` on the Go API.
2. On success, the Next.js route sets two cookies:
   - `rogele_access_token` (httpOnly, short-lived)
   - `rogele_refresh_token` (httpOnly, long-lived)
   It also sets a non-sensitive `rogele_user` cookie with the user's id, email,
   name, and role for server-side rendering.
3. Middleware (`src/middleware.ts`) redirects unauthenticated requests to
   `/login` and authenticated users away from the auth pages.
4. Server components use `src/lib/server-api.ts`, which reads the access token
   cookie and injects it as an `Authorization` header when calling the Go API.
5. Client components call `/api/proxy/...`, which forwards to the API with the
   cookie token attached.

Refreshing happens automatically when the access token expires — call
`/api/auth/refresh` from the client, or send a request and let the middleware
handle the redirect.

## Branding and Theming

Branding is loaded from the API at runtime. The server fetches
`GET /api/v1/config/public` on every page render, which returns:

    {
      "branding": {
        "school_name": "Rays of Grace",
        "tagline": "E-Learning",
        "logo_url": "",
        "favicon_url": "",
        "color_primary": "#7f2677",
        "color_accent": "#f06724",
        "support_email": "",
        "website_url": ""
      }
    }

Two things happen with these values:

1. `BrandingProvider` (`src/components/brand/brand-provider.tsx`) publishes
   them via React context and sets CSS variables `--brand-primary` and
   `--brand-accent` on the document root. Every component that uses
   `text-brand-primary`, `bg-brand-accent`, and similar utilities inherits the
   values automatically.
2. Next.js metadata uses the school name and favicon for the page title and
   browser tab icon.

Admins edit branding at `/admin/settings`. Changes take effect within 60
seconds (the API caches the branding block).

The theme (light, dark, or system) is stored in `localStorage` under
`rogele.theme` and applied before first paint by an inline script in the root
layout. There is no flash of the wrong theme.

## Notifications

Every authenticated user has a notification bell in the header. The bell:

- Shows an unread count fetched from
  `GET /api/v1/notifications/me/unread-count`.
- Polls every 30 seconds to refresh the count.
- Opens a dropdown with the latest 10 notifications.
- Navigates to the notification's link, marking it read, when clicked.
- Offers "Mark all read" inside the dropdown.

The full list lives at `/notifications` with filters by type and unread-only.

The admin bell's "View all" link goes to `/admin/notifications`, which provides
a composer for broadcasts and a history of everything sent.

## API Communication

All API calls go through one of two paths:

- **Server components** use `serverApiRequest` from `src/lib/server-api.ts`.
  This reads the access token cookie and injects the `Authorization` header.
- **Client components** use one of the `*.client.ts` helpers, which call
  `/api/proxy/...`. The proxy route at
  `src/app/api/proxy/[...path]/route.ts` reads the same cookie, forwards the
  request to the Go API, and returns the response body.

The proxy exports handlers for `GET`, `POST`, `PUT`, `PATCH`, `DELETE`,
`HEAD`, and `OPTIONS`. If a new HTTP method is needed, add it to the exports.

Non-JSON responses are handled defensively. If the API returns plain text (for
example, a 404 body), the client wraps it as an error message instead of
throwing a JSON parse exception.

## Roles and Routes

Route access is enforced by two layers of layout guards.

The `(app)` group requires a session, and redirects admins to `/admin`. It
wraps:

- `/dashboard`
- `/lessons`
- `/homework`
- `/quizzes`
- `/subscription`
- `/profile`
- `/notifications`

The `(admin)` group requires `role === 'admin'`, and redirects non-admins to
`/dashboard`. It wraps every page under `/admin`.

`/login`, `/register`, `/forgot-password`, and `/reset-password` are public.
Middleware redirects signed-in users away from them.

## Development Workflow

Common commands:

    # Start the dev server
    pnpm dev

    # Type-check the whole project
    pnpm type-check

    # Lint
    pnpm lint

    # Format with Prettier
    pnpm format

    # Build for production
    pnpm build

    # Run the production build
    pnpm start

Before committing, make sure:

    pnpm type-check
    pnpm build

Both should pass with no errors.

## Building for Production

    pnpm build

The build produces a `.next` directory with the compiled app. Serve it with:

    pnpm start

Or place the output behind a reverse proxy (nginx, Caddy, Cloudflare) that
terminates TLS and forwards to `localhost:3000`.

Environment variables needed at runtime:

- `ROGELEDB_API_URL` — the URL of the Go API
- `SESSION_SECRET` — must match between builds if cookies are expected to keep
  working across deploys
- `NEXT_PUBLIC_APP_URL` — the public URL of this app

## Deployment

The app is designed to run as a Node.js process. Recommended deployments:

- **Vercel** — zero-config for Next.js. Set the environment variables in the
  project settings. Note that `ROGELEDB_API_URL` must be reachable from Vercel's
  servers, so the API cannot be on a private network.
- **Docker + a VPS** — build a Docker image that runs `pnpm start` and expose
  port 3000. Put Cloudflare or Caddy in front for TLS.
- **Railway, Render, Fly.io** — all support Node.js processes and environment
  variables.

Cookies are set with `secure: true` when `NODE_ENV=production`, so the app must
be served over HTTPS in production.

## License

This project is proprietary. All rights reserved.