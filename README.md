# OmniMed AI - Core Backend & AI Microservices 🏥

The distributed backend infrastructure for OmniMed AI, an automated multi-disease triage and diagnostic ecosystem. This repository contains the Go API gateway and the Python FastAPI/gRPC inference engine, starting with breast cancer ultrasound classification (skin and oral cancer follow the same pattern).

Designed like a CTO. Executed like an engineer. Shipped like a founder.

## 🏗️ System Architecture

```text
┌───────────────────────────┐
│      Client (mobile)       │
└──────────┬───────┬─────────┘
           │       │
           │       └────────────────────────┐
           ▼ (multipart/form-data over HTTP) ▼ (gRPC, reserved — auth/session/medical logs)
┌─────────────────────────┐        ┌─────────────────────────┐
│  Go Gateway — HTTP :8080 │        │ Go Gateway — gRPC :9090 │───► [ PostgreSQL ] (placeholder)
│  image-upload diagnosis  │        │ health service only so   │
│  + health/readiness only │        │ far — business services  │
└────────────┬─────────────┘        │ register on as built     │
             │                      └─────────────────────────┘
             ▼ (gRPC / protobuf — TriageService, client role)
┌─────────────────────────┐
│  Python AI Inference     │
│  services/ai-inference   │
│  (FastAPI health + gRPC) │
└────────────┬────────────┘
             │
             ▼
┌─────────────────────────┐
│  ONNX Runtime            │
│  omnimed_breast_ultrasound.onnx │
└─────────────────────────┘
```

### Component Breakdown

1. **Go Gateway (`cmd/gateway`)** — runs two servers concurrently, mirroring the inference service's own HTTP-health/gRPC-business split:
   - **HTTP (`:8080`)** — the *only* thing HTTP serves is image-upload diagnosis (`POST /api/v1/diagnose/breast-cancer`, needs `multipart/form-data`) plus `/healthz` and `/readyz`. It forwards the image to the AI inference service over gRPC, applies the decision policy, and returns JSON.
   - **gRPC (`:9090`, `internal/gateway/delivery/grpc`)** — reserved for every other client-facing operation as it's built: auth, session, medical logs. Only the standard health service is registered on it today.
   Auth is currently a stubbed pass-through middleware (`internal/gateway/delivery/http/middleware`), structured so real JWT/RBAC drops in without touching routes or handlers.
2. **Python AI Inference Service (`services/ai-inference`)** — loads `omnimed_breast_ultrasound.onnx` with ONNX Runtime, implements the `TriageService` gRPC contract (called by the gateway's HTTP handler, not directly by clients), and handles image preprocessing (resize to 224x224, ImageNet mean/std normalization).
3. **PostgreSQL** — present in `docker-compose.yml` for local dev, with a placeholder connection config in `internal/config`. No repository uses it yet; there's no persistence need until user accounts / medical logs are added.

Adding `POST /api/v1/diagnose/skin` or `/oral` later is a small addition, not a redesign: a thin wrapper handler (`h.handleDiagnose(w, r, "skin", h.usecase.DiagnoseSkin)`), one route line in `router.go`, and the corresponding proto RPC + usecase method — the parsing/validation/response-writing skeleton in `handler.go` is already shared.

## 📁 Repository Layout

```text
api/proto/triage.proto        # TriageService contract (shared by Go + Python)
cmd/gateway/                  # Go gateway entrypoint + Dockerfile
internal/
  gateway/
    delivery/http/            # router, handlers, stub auth middleware (image diagnosis + health)
    delivery/grpc/            # gateway's own gRPC server (health only today; auth/session/logs later)
    usecase/                  # request validation, orchestration, decision policy
    repository/grpc/          # gRPC client to the inference service
  config/                     # Viper-based env config (+ placeholder Postgres DSN)
  logger/                     # structured (slog) JSON logging
pb/triage/                    # generated Go protobuf/gRPC stubs
services/ai-inference/
  app/
    main.py                   # entrypoint: starts gRPC server + health FastAPI app
    config.py                 # pydantic-settings env config
    logging_config.py         # structured JSON logging
    grpc_server.py            # TriageService servicer implementation
    inference/                # ONNX model wrapper + preprocessing
    pb/                       # generated Python protobuf/gRPC stubs
  requirements.txt
  Dockerfile
models/
  omnimed_breast_ultrasound.onnx   # 3-class (benign/malignant/normal) ONNX model
docker-compose.yml             # gateway + ai-inference + postgres
Makefile                       # proto generation, build, run, docker shortcuts
```

## 📊 Diagnostic Capabilities & Modalities

| Target Pathology | Clinical Modality / Input | Diagnostic Approach | Status |
| :--- | :--- | :--- | :--- |
| **Breast Cancer** | Uploaded Ultrasound Scan | Classifies benign / malignant / normal via ONNX CNN | ✅ Implemented |
| **Skin Cancer** | Surface Macroscopic Photo | Lesion shape, border, and texture analysis | 🔜 Planned — same `TriageService` pattern (`DiagnoseSkin`) |
| **Oral Cancer** | High-Flash Intrabuccal Photo | Screens for ulcers, leukoplakia, mucosal lesions | 🔜 Planned — same `TriageService` pattern (`DiagnoseOral`) |

Adding a new modality means adding a new RPC to `TriageService` (e.g. `DiagnoseSkin(ImageRequest) returns (DiagnosisResponse)`) — the shared `ImageRequest`/`DiagnosisResponse` messages mean this is additive and doesn't break existing clients.

## 🛠️ Backend Tech Stack

* **Languages:** Go (1.25+), Python (3.12+)
* **Communication:** gRPC over HTTP/2, Protocol Buffers (proto3)
* **Frameworks:** net/http + Viper (Go), FastAPI + ONNX Runtime + pydantic-settings (Python)
* **Infrastructure:** PostgreSQL, Docker & Docker Compose

## 🚀 Local Development Setup

### 1. Copy the env file

```bash
cp .env.example .env
```

### 2. Compile Protocol Buffers

```bash
make proto        # both Go and Python stubs
make proto-go      # Go only
make proto-python  # Python only (requires services/ai-inference/.venv, see `make build-inference`)
```

### 3. Run natively (no Docker)

```bash
make build-inference   # creates the Python venv and installs dependencies
make run-inference      # starts the gRPC + health server on :50051 / :8000

make run-gateway        # in a second terminal, starts the gateway: HTTP :8080, gRPC :9090
```

### 4. Or run everything with Docker Compose

```bash
make docker-up
# gateway:      http://localhost:8080/healthz  (business gRPC on :9090, currently health-only)
# ai-inference: http://localhost:8000/healthz  (gRPC on :50051)
# postgres:     localhost:5432
```

### 5. Confirm the whole stack is actually up

`/healthz` only tells you the gateway process itself started — it says nothing about whether it can actually serve a diagnosis. `/readyz` checks that too, by calling the inference service's gRPC health endpoint (the standard `grpc.health.v1` protocol):

```bash
curl http://localhost:8080/readyz
```

```json
{ "status": "ready", "dependencies": { "ai_inference": "ok" } }
```

If the inference service is down or unreachable, this returns HTTP `503` instead, with the reason in `dependencies.ai_inference` — useful whether you started everything with `docker compose up` or ran the two processes separately in your own terminals and want to confirm they actually found each other.

### 6. Try it

```bash
curl -F "image=@sample_ultrasound.png" http://localhost:8080/api/v1/diagnose/breast-cancer
```

```json
{
  "risk_tier": "likely_benign",
  "raw_probabilities": {
    "benign": 0.91,
    "malignant": 0.05,
    "normal": 0.04
  },
  "confidence": 0.91,
  "model_version": "omnimed-breast-ultrasound-v1.0.0",
  "disclaimer": "This is an AI screening aid, not a medical diagnosis. All results must be reviewed and confirmed by a qualified clinician before any clinical decision is made.",
  "request_id": "..."
}
```

`risk_tier` is produced by a decision policy (`internal/gateway/usecase/decision_policy.go`), not raw argmax: a `malignant` probability at or above `MALIGNANT_FLAG_THRESHOLD` (default `0.179`, tuned from validation data) always yields `flagged_for_review`, even if it isn't the top class. Otherwise, if the best remaining probability is below `MIN_CONFIDENCE_FLOOR` (default `0.4`), the result is `inconclusive` rather than a false-confident `likely_benign`. Both thresholds are env-configurable — see `.env.example`.

---
*Developed as a Final Year Project at The ICT University.*
