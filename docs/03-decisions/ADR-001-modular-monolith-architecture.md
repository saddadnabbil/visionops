# ARCHITECTURE DECISION RECORD (ADR)
## ADR-001: Modular Monolith vs Microservices Architecture

| Field | Value |
| :--- | :--- |
| **ADR ID** | ADR-001 |
| **Status** | Accepted |
| **Decision Owner** | Nabbil / Lead Architect |
| **Decision Date** | September 2026 |
| **Related Docs** | [ARCHITECTURE-OVERVIEW.md](../02-architecture/ARCHITECTURE-OVERVIEW.md) |

---

## 1. Context & Problem Statement
VisionOps membutuhkan sistem backend yang cepat, efisien, dan memiliki konsistensi data tinggi saat menghubungkan deteksi AI, pembuatan tiket insiden, pencatatan log audit K3, dan pengiriman notifikasi webhook.

Pilihan arsitektur yang dipertimbangkan:
1. **Microservices** (Layanan Ingest terpisah, Alerting terpisah, Webhook terpisah dengan Kafka/RabbitMQ).
2. **Modular Go Monolith** (Single binary dengan internal domain packages yang terisolasi dan database PostgreSQL bersama).

## 2. Decision
Kami memilih **Modular Go Monolith** dengan komponen internal yang terbagi jelas (`internal/ingest`, `internal/incidents`, `internal/outbox`, `internal/auth`, `internal/audit`).

## 3. Rationale & Alternatives Considered

| Kriteria | Modular Go Monolith (Dipilih) | Microservices + Message Broker |
| :--- | :--- | :--- |
| **Konsistensi Transaksi (ACID)** | **Sangat Tinggi**: Dapat commit deteksi, insiden, audit, dan outbox dalam 1 DB transaction. | **Rendah/Kompleks**: Memerlukan 2-phase commit atau saga pattern terdistribusi. |
| **Operational Overhead** | **Sangat Rendah**: Cukup 1 binary Go + 1 instance DB. Mudah di-deploy di VPS murah atau single Docker container. | **Tinggi**: Perlu mengelola cluster Kafka, Kubernetes/multiple containers, tracing terdistribusi. |
| **Latensi Ingestion** | **Sub-millisecond**: Pemanggilan fungsi in-memory antar modul Go. | **Network Hop**: Ada overhead serialisasi JSON/gRPC dan latency antrian. |
| **Kemudahan Refactoring** | **Tinggi**: Perubahan contract antar domain terdeteksi langsung oleh Go compiler. | **Sedang**: Perlu versioning API antar service. |

## 4. Consequences
- **Positive**:
  - Development speed sangat cepat dan tidak ada overhead infrastruktur broker.
  - Zero risk inkonsistensi data antara insiden dan outbox queue.
- **Negative / Trade-offs**:
  - Semua modul berbagi resource CPU/RAM yang sama pada host.
  - *Mitigasi*: Beban outbox worker dapat dipisah menjadi sub-command / replica worker tersendiri jika beban antrian webhook meningkat di masa depan.
