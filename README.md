# AdOrbit Authentication

AdOrbit Authentication is a Go microservice for authentication and session management. It uses PostgreSQL to store credentials and sessions, and includes an internal gRPC service for access-token validation.

> **Development status:** The HTTP routes are registered, but their handlers are not fully implemented yet. Signup currently validates its request body; login, logout, and refresh are placeholders. Do not rely on these endpoints for production authentication yet.

## Requirements

- Go 1.27.1
- Docker and Docker Compose (for the included PostgreSQL setup)

## Configuration

The service reads configuration from environment variables. A `.env` file in the project root is loaded automatically when present.

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `DATABASE_URL` | Yes | — | PostgreSQL connection URL |
| `APP_PORT` | No | `8080` | HTTP server port |
| `POSTGRES_USER` | For Docker Compose | — | PostgreSQL user |
| `POSTGRES_PASSWORD` | For Docker Compose | — | PostgreSQL password |
| `POSTGRES_DB` | For Docker Compose | — | PostgreSQL database name |

For local development with the included Compose database, create a `.env` file in the project root:

```dotenv
POSTGRES_USER=adorbit
POSTGRES_PASSWORD=change-me
POSTGRES_DB=adorbit_auth
DATABASE_URL=<PostgreSQL URL for localhost:5434/adorbit_auth>
APP_PORT=8080
```

Replace the `DATABASE_URL` placeholder with a PostgreSQL connection URL using the credentials above. URL-encode the username or password if either contains reserved URL characters.

## Run locally

Start PostgreSQL:

```sh
docker compose up -d postgres
```

Then, from the project root, start the service:

```sh
go run ./cmd/server
```

The service applies database migrations on startup and listens on `http://localhost:8080` by default. Change `APP_PORT` to use another port.

To stop the database container:

```sh
docker compose down
```

The database data is stored in the Compose-managed `postgres_data` volume.

## HTTP endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/health` | Health check |
| `POST` | `/api/v1/auth/signup` | Signup (request validation only at present) |
| `POST` | `/api/v1/auth/login` | Login (not implemented) |
| `POST` | `/api/v1/auth/logout` | Logout (not implemented; route requires authentication) |
| `POST` | `/api/v1/auth/refresh` | Refresh tokens (not implemented; route requires authentication) |

Health check example:

```sh
curl http://localhost:8080/health
```

Expected response:

```json
{"status":"ok"}
```

The internal gRPC server listens on `127.0.0.1:9090` and is used for access-token validation. It is not exposed as a public service endpoint.

## Docker image

Build the service image from the project root:

```sh
docker build -t adorbit-authentication .
```

The image exposes HTTP port `8080`. Supply `DATABASE_URL` and, if needed, `APP_PORT` when running the container.
