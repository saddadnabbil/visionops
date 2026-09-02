package cameraadapter

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientDiscoversNamedTenantCamera(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ingest/cameras" || r.Method != http.MethodGet {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "adapter-key" {
			t.Fatalf("missing adapter key")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"camera-a","name":"Line A Entrance","location":"Factory"},{"id":"camera-b","name":"Line B Entrance","location":"Factory"}]`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "adapter-key", server.Client())
	camera, err := client.DiscoverCamera(context.Background(), "Line B Entrance")
	if err != nil {
		t.Fatal(err)
	}
	if camera.ID != "camera-b" || camera.Name != "Line B Entrance" {
		t.Fatalf("unexpected camera: %#v", camera)
	}
}

func TestClientSendsHeartbeatAndDetectionUsingProducerContract(t *testing.T) {
	requests := []struct {
		Path string
		Body string
	}{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, struct{ Path, Body string }{r.URL.Path, string(body)})
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"accepted"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "adapter-key", server.Client())
	if err := client.Heartbeat(context.Background(), "camera-a"); err != nil {
		t.Fatal(err)
	}
	detection := Detection{EventID: "fixture-1", CameraID: "camera-a", Rule: "missing_ppe", Severity: "high", ObservedAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), Metadata: map[string]any{"source": "fixture"}}
	if err := client.SendDetection(context.Background(), detection); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 || requests[0].Path != "/api/v1/ingest/camera-heartbeats" || requests[1].Path != "/api/v1/ingest/detections" {
		t.Fatalf("unexpected requests: %#v", requests)
	}
	if !bytes.Contains([]byte(requests[0].Body), []byte(`"camera_id":"camera-a"`)) || !bytes.Contains([]byte(requests[1].Body), []byte(`"event_id":"fixture-1"`)) {
		t.Fatalf("unexpected request payloads: %#v", requests)
	}
}
