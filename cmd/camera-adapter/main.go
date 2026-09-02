// camera-adapter is a legal, fixture-driven stand-in for an approved AI Camera
// Service. It sends only the narrow VisionOps producer contract; it does not
// access, scrape, persist, or display public CCTV video.
package main

import (
	"context"
	"fmt"
	"github.com/nabbil/visionops/internal/cameraadapter"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key, fallback string) time.Duration {
	value, err := time.ParseDuration(env(key, fallback))
	if err != nil || value <= 0 {
		return mustDuration(fallback)
	}
	return value
}

func mustDuration(value string) time.Duration {
	duration, _ := time.ParseDuration(value)
	return duration
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	apiURL := env("VISIONOPS_API_URL", "http://localhost:18080")
	apiKey := os.Getenv("VISIONOPS_API_KEY")
	if apiKey == "" {
		log.Error("VISIONOPS_API_KEY is required")
		os.Exit(1)
	}
	scenarioPath := env("CAMERA_ADAPTER_SCENARIO", "fixtures/camera-adapter/missing-ppe.json")
	scenario, err := cameraadapter.LoadScenario(scenarioPath)
	if err != nil {
		log.Error("invalid adapter scenario", "path", scenarioPath, "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client := cameraadapter.NewClient(apiURL, apiKey, nil)
	camera := discover(ctx, log, client, env("CAMERA_ADAPTER_CAMERA_NAME", ""))
	if camera.ID == "" {
		return
	}
	log.Info("camera adapter connected", "camera", camera.Name, "location", camera.Location, "scenario", filepath.Base(scenarioPath))
	go heartbeatLoop(ctx, log, client, camera.ID, durationEnv("CAMERA_ADAPTER_HEARTBEAT_INTERVAL", "30s"))
	for index, event := range scenario.Events {
		if !wait(ctx, time.Duration(event.AfterSeconds)*time.Second) {
			return
		}
		detection := cameraadapter.Detection{
			EventID:    fmt.Sprintf("fixture-%d-%d", time.Now().UTC().UnixNano(), index+1),
			CameraID:   camera.ID,
			Rule:       event.Rule,
			Severity:   event.Severity,
			ObservedAt: time.Now().UTC(),
			Metadata: merge(merge(event.Metadata, scenario.Metadata()), map[string]any{
				"producer": "fixture-camera-adapter",
				"scenario": filepath.Base(scenarioPath),
			}),
		}
		if err := client.SendDetection(ctx, detection); err != nil {
			log.Error("detection delivery failed", "rule", event.Rule, "error", err)
			continue
		}
		log.Info("scenario detection accepted", "mode", scenario.Mode, "rule", event.Rule, "severity", event.Severity)
	}
	<-ctx.Done()
}

func discover(ctx context.Context, log *slog.Logger, client *cameraadapter.Client, name string) cameraadapter.Camera {
	for {
		camera, err := client.DiscoverCamera(ctx, name)
		if err == nil {
			return camera
		}
		log.Warn("camera discovery retrying", "error", err)
		if !wait(ctx, 3*time.Second) {
			return cameraadapter.Camera{}
		}
	}
}

func heartbeatLoop(ctx context.Context, log *slog.Logger, client *cameraadapter.Client, cameraID string, interval time.Duration) {
	send := func() {
		if err := client.Heartbeat(ctx, cameraID); err != nil {
			log.Warn("camera heartbeat failed", "error", err)
			return
		}
		log.Info("camera heartbeat accepted", "camera_id", cameraID)
	}
	send()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			send()
		case <-ctx.Done():
			return
		}
	}
}

func wait(ctx context.Context, duration time.Duration) bool {
	if duration == 0 {
		return true
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func merge(first, second map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range first {
		result[key] = value
	}
	for key, value := range second {
		result[key] = value
	}
	return result
}
