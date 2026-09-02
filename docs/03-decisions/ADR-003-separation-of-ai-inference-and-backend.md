# ARCHITECTURE DECISION RECORD (ADR)
## ADR-003: Decoupling AI Computer Vision Inference from Operational Backend

| Field | Value |
| :--- | :--- |
| **ADR ID** | ADR-003 |
| **Status** | Accepted |
| **Decision Owner** | Nabbil / Lead Architect |
| **Decision Date** | September 2026 |
| **Related Docs** | [PRD-VisionOps.md](../01-product/PRD-VisionOps.md) |

---

## 1. Context & Problem Statement
Aplikasi safety monitoring membutuhkan model computer vision (seperti YOLOv8 / PyTorch) untuk menganalisis frame video RTSP dari kamera IP. Timbul pertanyaan apakah model inference harus di-embed langsung di dalam aplikasi Go utama atau dipisahkan sebagai Edge Service.

## 2. Decision
Kami memutuskan untuk **memisahkan secara ketat (Decouple)** AI Computer Vision Inference Service dari VisionOps Core Backend melalui kontrak API standar (`POST /api/v1/ingest/detections`).

## 3. Rationale
1. **Karakteristik Resource Berbeda**: AI Inference membutuhkan GPU, library Python/C++, dan konsumsi memori tinggi. VisionOps Go Backend adalah layanan I/O-bound ringan yang membutuhkan CPU rendah dan throughput transaksi cepat.
2. **Hardware Agnostic**: Backend dapat di-deploy di cloud server murah manapun tanpa ketergantungan driver NVIDIA CUDA.
3. **Resilience & Fault Isolation**: Jika frame grabber kamera RTSP crash atau model AI hang karena overload video, sistem insiden dan dashboard pos kontrol tetap 100% online untuk operator manusia.
4. **Model Pluggability**: Tim AI dapat meng-upgrade model deteksi (misal dari YOLOv8 ke YOLOv11) atau mengganti vendor kamera tanpa perlu menyentuh atau me-redeploy kode backend VisionOps.
