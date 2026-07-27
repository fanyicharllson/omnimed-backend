# OmniMed AI - Core Backend & AI Microservices 🏥

The distributed backend infrastructure for OmniMed AI, an automated multi-disease triage and diagnostic ecosystem. This repository contains the high-concurrency Go API gateway, the event-driven RabbitMQ messaging topology, and the high-performance Python FastAPI/gRPC inference engines.

Designed like a CTO. Executed like an engineer. Shipped like a founder.

## 🏗️ System Architecture

OmniMed bypasses traditional monolithic REST bottlenecks by utilizing an event-driven, decoupled microservices paradigm optimized for high scalability and low-resource edge deployment.



┌─────────────────────────┐
│ Flutter Mobile Client │
└────────────┬────────────┘
│
▼ (gRPC / HTTP/2)
┌─────────────────────────┐
│ Go Gateway Service │────► [ PostgreSQL ]
└────────────┬────────────┘
│
┌──────────────┴──────────────┐
│ │
▼ (Direct gRPC Stream) ▼ (Publish Event)
┌─────────────────────────┐ ┌─────────────────────────┐
│ Python AI Service (ONNX)│ │ RabbitMQ Job Queue │
│ (FastAPI Framework) │ └─────────────────────────┘
└─────────────────────────┘

### Component Breakdown
1. **Core Orchestrator (Go):** Handles user session state, strict Role-Based Access Control (RBAC) via JWTs, medical log history, and acts as the entry reverse-proxy.
2. **AI Inference Service (Python/FastAPI):** A high-performance gRPC worker wrapper that accepts raw binary image streams, reconstructs image tensors, and passes them to optimized ONNX models.
3. **Message Broker (RabbitMQ):** Ensures eventual consistency and system resilience during high clinical loads by queueing heavy data diagnostic pipelines asynchronously.

## 📊 Diagnostic Capabilities & Modalities

| Target Pathology | Clinical Modality / Input | Diagnostic Approach | Dataset Core |
| :--- | :--- | :--- | :--- |
| **Skin Cancer** | Surface Macroscopic Photo | Analyzes lesion shape asymmetry, irregular borders, and color textures. | HAM10000 / ISIC |
| **Oral Cancer** | High-Flash Intrabuccal Photo | Screens for mouth ulcers, leukoplakia (white patches), and mucosal lesions. | Oral Cancer Dataset |
| **Breast Cancer** | Uploaded Ultrasound Scan / Mammogram | Segments deep tissue architecture to classify anomalies as benign or malignant. | BUSI Dataset |

## 🛠️ Backend Tech Stack

* **Languages:** Go (v1.22+), Python (v3.11+)
* **Communication Protocols:** gRPC over HTTP/2, Protocol Buffers (Proto3)
* **Frameworks:** FastAPI, ONNX Runtime
* **Databases & Infrastructure:** PostgreSQL, RabbitMQ, Docker & Docker Compose

## 🚀 Local Development Setup

### 1. Compile Protocol Buffers
From the root directory, compile your `.proto` files into native Go schemas and Python stubs:
```bash
# Compile for Go
protoc --go_out=. --go-grpc_out=. proto/triage.proto

# Compile for Python
python -m grpc_tools.camel_case_to_lower_strict --python_out=. --grpc_python_out=. proto/triage.proto
```

### 2. Launch Local Environment (Docker Compose)
Simulate the production environment locally on your development system using a lightweight containerized stack:
```bash
docker-compose up -d --build
```

---
*Developed as a Final Year Project at The ICT University.*

