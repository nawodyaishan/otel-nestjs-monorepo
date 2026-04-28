# Tech Spec: NestJS + Vite + Turborepo (Platform Observability Testing)

## 🎯 Goal
Create a minimal monorepo to test OpenTelemetry (OTel) tracing, logging, and metrics using:
* **NestJS** (Backend)
* **Vite** (Frontend)
* **OpenTelemetry Collector Contrib**

The system is designed to establish foundational DevOps and Platform Engineering practices. It will generate distributed traces, correlated logs, and metrics from the frontend to the backend, eventually collected and visualized via an observability stack.

---

## 🏗️ Architecture & Stack

### Core Technologies
- **Monorepo Manager:** Turborepo
- **Package Manager:** pnpm (using `pnpm-workspace.yaml`)
- **Backend:** NestJS, `@opentelemetry/sdk-node`, `@opentelemetry/auto-instrumentations-node`
- **Frontend:** Vite + React (TypeScript)
- **Observability Stack:** OpenTelemetry Collector, Jaeger (Tracing), Prometheus (Metrics), Grafana (Visualization)
- **Containerization & Orchestration:** Docker, Docker Compose

### Directory Structure
```text
repo-root/
├── apps/
│   ├── api/                # NestJS application (Port 3000)
│   │   └── Dockerfile      # Multi-stage optimized build for API
│   └── web/                # Vite React application (Port 5173 / 80)
│       └── Dockerfile      # Multi-stage optimized build for Web
├── packages/
│   └── config/             # Shared ESLint/TS configs
├── otel/
│   ├── otel-collector.yaml # OTel Collector Configuration
│   ├── prometheus.yml      # Prometheus metrics scraping config
│   └── grafana/            # Grafana dashboard provisioning
├── docker-compose.yml      # Orchestrates api, web, otel-collector, jaeger, prometheus, grafana
├── package.json            # Root workspace definitions
├── pnpm-workspace.yaml     # pnpm workspace config
└── turbo.json              # Turborepo pipeline configuration
```

---

## 🔍 Observability & Monitoring Strategy

To adhere to Platform Engineering best practices, we will implement the **Three Pillars of Observability**:

1. **Distributed Tracing (Jaeger + OTel):** 
   - Every request is assigned a `trace_id`. Context is propagated via HTTP Headers (`traceparent`).
   - Implement **Probabilistic Sampling** (e.g., 100% for dev, 10% for prod).
   - Use Span tags/attributes (e.g., `user.id`, `http.status_code`) to enable rich querying.

2. **Correlated Logging:** 
   - Use a structured logger (like Pino or Winston) in NestJS.
   - Inject the active `trace_id` and `span_id` into every log entry so logs can be cross-referenced with traces in Grafana/Jaeger.

3. **Metrics (Prometheus):**
   - Expose RED metrics (Rate, Errors, Duration) for the NestJS API.
   - The OTel Collector will export metrics to Prometheus.

---

## 🚀 Implementation Phases

### Phase 1: Foundation Setup (Turborepo + pnpm)
**Objective:** Initialize the monorepo structure and Turborepo configuration with pnpm.
1. Initialize a new Git repository and standard `.gitignore`.
2. Create `pnpm-workspace.yaml` defining `apps/*` and `packages/*`.
3. Set up `turbo.json` with standard build and dev pipelines.

### Phase 2: Comprehensive Observability Infrastructure
**Objective:** Stand up the OTel collector, Jaeger, Prometheus, and Grafana.
1. Create `otel/otel-collector.yaml`.
2. Configure **Receivers**: OTLP over HTTP (4318) and gRPC (4317).
3. Configure **Processors**: Batch processor, memory limiter.
4. Configure **Exporters**: 
   - `jaeger` or `otlp` exporting to a local Jaeger instance.
   - `prometheus` exporting to a local Prometheus instance.
   - `logging` for debug output.
5. Define `docker-compose.yml` to include `otel-collector`, `jaeger`, `prometheus`, and `grafana`.

### Phase 3: Backend Development & Instrumentation (NestJS API)
**Objective:** Build a NestJS API that emits traces, metrics, and correlated logs.
1. Scaffold NestJS in `apps/api`.
2. Install OpenTelemetry Node.js SDK and auto-instrumentations.
3. Create `src/tracer.ts` to initialize the OTel SDK pointing to `http://otel-collector:4318/v1/traces`.
4. Implement **Correlated Logging**: Configure the NestJS logger to extract the active `trace_id` from the OTel context and append it to JSON log output.
5. Import `tracer.ts` at the absolute top of `main.ts`.
6. Expose a `GET /hello` endpoint that simulates business logic, adding custom span events and attributes (e.g., `span.setAttribute('user.filter', 'active')`).

### Phase 4: Frontend Development (Vite)
**Objective:** Build a simple frontend to trigger the backend API.
1. Scaffold a Vite + React app inside `apps/web`.
2. Create a UI with a "Fetch Hello" button.
3. Wire the button to make a fetch request to the backend.
4. Integrate `@opentelemetry/instrumentation-document-load` and `@opentelemetry/instrumentation-fetch` to propagate the `traceparent` header to the backend.

### Phase 5: Full Dockerization & Orchestration
**Objective:** Containerize the applications securely and optimally.
1. **NestJS Dockerfile**: Multi-stage build, dropping root privileges (`USER node`), optimizing layer caching.
2. **Vite Dockerfile**: Multi-stage build, serving built static files using an unprivileged Nginx alpine image.
3. **Docker Compose**: Wire `api` and `web` into the existing observability network, ensuring proper startup order (`depends_on`).

---

## ✅ Success Criteria
- [ ] Entire platform (Apps + Observability Stack) starts with `docker compose up --build`.
- [ ] Frontend button click results in an end-to-end trace viewable in Jaeger.
- [ ] NestJS stdout logs are JSON structured and include `trace_id` and `span_id`.
- [ ] OTel Collector successfully routes signals to both Jaeger (Traces) and Prometheus (Metrics).
- [ ] Grafana connects to Prometheus/Jaeger allowing unified visualization.
