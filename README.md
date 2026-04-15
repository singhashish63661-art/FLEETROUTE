<<<<<<< HEAD

=======
# GPS Fleet Management Platform

An enterprise-grade, multi-tenant GPS tracking and fleet management platform written in Go (Backend) and React/TypeScript (Frontend). It features real-time vehicle tracking, geofencing, advanced trip analytics, and an event-driven architecture built for high throughput.

## System Architecture

The platform uses a microservices architecture communicating through **NATS JetStream** for event streams and **Redis** for state, and relying on **TimescaleDB** (PostgreSQL extension) for high-performance time-series data storage.

### Services Structure
- **`ingestion-service`**: Listens on raw TCP/UDP ports (e.g., JT/T808, Teltonika formats). It parses Hex protocols and streams standard JSON payloads into NATS JetStream.
- **`stream-processor`**: The intelligent engine. Pulls raw coordinates from NATS JetStream, executes enrichment pipelines (Trip State FSM, Geofencing Checks, Alert Rule Evaluations), and writes the final structured analytical records into TimescaleDB.
- **`api-service`**: Serves conventional REST APIs for User Management, Fleet CRUD, Reporting Analytics, and Driver Scoring.
- **`websocket-service`**: Handles real-time WebSocket connections spanning thousands of clients to push direct location updates directly to browser instances via Redis Pub/Sub.
- **`frontend`**: A Vite-powered React Single-Page Application featuring real-time maps, Recharts-based reporting, and multi-tenant fleet administration.

### Tech Stack
- **Languages**: Go 1.22, TypeScript, SQL
- **Messaging**: NATS JetStream
- **Cache/State**: Redis 7
- **Database**: PostgreSQL 15 + TimescaleDB + PostGIS
- **UI Framework**: React, Vite, Recharts, Lucide Icons

---

## 🚀 Quick Start / Running Instructions

### Prerequisites
- Docker and Docker Compose installed
- Make utility (for generating secrets)
- Go (if compiling local binaries outside docker)

### 1. Generating Security Keys
The platform requires a private/public key pair (RS256) to cryptographically sign JWTs. Run the generator script first:
```bash
make gen-keys
```
*(This guarantees your \`secrets/\` directory holds the keys required by the api-service configuration.)*

### 2. Spinning Up Data Infrastructure
Bring up TimescaleDB, Redis, and NATS JetStream:
```bash
# Deploys backend dependencies and application containers
docker compose up -d --build
```

### 3. Database Initial Seed
You need to seed the freshly created PostgreSQL database with a super-admin account to be able to log in to the frontend.
```bash
# Execute the seed script
go run scripts/seed_admin.go
```
This generates the initial root tenant and an administrative user with the following credentials:
- **Email:** `admin@gpsgo.com`
- **Password:** `admin123`

### 4. Viewing the Application
The environment is localized and proxy-routed:
- **Frontend Dashboard**: `http://localhost:3000`
- **Grafana Metrics**: `http://localhost:3001`

---

## Modifying & Developing 

If taking development locally out of Docker Compose mode:

### Building and Formatting
For Go services, from the project root:
```bash
go build ./...
```
For the Frontend, ensure NodeJS 20+ is installed:
```bash
cd frontend
npm install
npm run dev
```

### Applying Migrations
Database tables are strictly modeled traversing `/migrations/`. The standard \`golang-migrate/migrate\` tool applies them. In the standard `docker compose`, the `migrator` container will attempt to automatically run on spin-up. 

### Kubernetes Deployment
Production-grade configurations including Horizontal Pod Autoscalers, ConfigMaps, and Ingress NetworkPolicies exist in the `infra/k8s/` tree hierarchy mapping to this exact structure.

## Observability
A Prometheus target configuration operates to scrape `:metrics` exposed endpoints on `ingestion-service` and `stream-processor`, populating the native Grafana dashboards pre-configured in `infra/grafana/dashboards/`.
>>>>>>> a192893 (updated files)
