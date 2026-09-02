#!/usr/bin/env bash
set -euo pipefail

camera_id="${1:?Usage: ./scripts/simulate.sh CAMERA_ID [missing_ppe|restricted_area|crowding]}"
rule="${2:-missing_ppe}"
event_id="sim-$(date +%s)"
curl --fail-with-body -X POST http://localhost:18080/api/v1/ingest/detections \
  -H 'Content-Type: application/json' -H 'X-API-Key: vo_demo_ingest' \
  --data "{\"event_id\":\"${event_id}\",\"camera_id\":\"${camera_id}\",\"rule\":\"${rule}\",\"severity\":\"high\",\"metadata\":{\"simulator\":true}}"
