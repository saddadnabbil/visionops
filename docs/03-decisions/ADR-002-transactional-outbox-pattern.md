# ARCHITECTURE DECISION RECORD (ADR)
## ADR-002: Transactional Outbox Pattern for Webhook Deliveries

| Field | Value |
| :--- | :--- |
| **ADR ID** | ADR-002 |
| **Status** | Accepted |
| **Decision Owner** | Nabbil / Lead Architect |
| **Decision Date** | September 2026 |
| **Related Docs** | [ARCHITECTURE-OVERVIEW.md](../02-architecture/ARCHITECTURE-OVERVIEW.md) |

---

## 1. Context & Problem Statement
Saat insiden keselamatan terdeteksi, VisionOps harus memberitahu webhook eksternal (Telegram, Slack, sistem tiket pelanggan). Jika API langsung memanggil HTTP webhook pihak ketiga sebelum atau sesudah menyimpan ke database, ada risiko:
- **Kasus A (Dual-write failure)**: HTTP request sukses, tapi database error saat commit $\rightarrow$ Alarm palsu terkirim, data tidak ada di dashboard.
- **Kasus B (Dual-write failure)**: Database commit sukses, tapi endpoint pihak ketiga timeout/down $\rightarrow$ Insiden tercatat di dashboard, tapi tim darurat tidak pernah menerima notifikasi.

## 2. Decision
Menerapkan **Transactional Outbox Pattern** menggunakan tabel database `outbox_jobs` dan background polling worker dengan `FOR UPDATE SKIP LOCKED`.

## 3. Mechanism
1. Handler API menyimpan data insiden dan menyisipkan baris tugas notifikasi ke tabel `outbox_jobs` dalam **satu transaksi database ACID**.
2. Worker background secara asynchronous mengambil baris job `pending`, menandatangani payload dengan secret HMAC-SHA256, dan mengirimkannya via HTTP client.
3. Setiap upaya pengiriman dicatat di `webhook_delivery_attempts`.
4. Jika endpoint tujuan gagal merespon (5xx / network timeout), worker menjadwalkan ulang dengan *exponential backoff* ($2^n$ detik). Jika gagal 5x berturut-turut, job ditandai sebagai `dead_letter` untuk diinvestigasi atau di-replay manual oleh admin.

## 4. Consequences
- **Positive**:
  - Jaminan *At-Least-Once Delivery* (tidak ada notifikasi hilang).
  - Ingestion API tetap super cepat karena tidak menunggu respon HTTP pihak ketiga (non-blocking).
- **Negative**:
  - Konsumen webhook harus idempoten (siap menerima event yang sama jika terjadi retry).
