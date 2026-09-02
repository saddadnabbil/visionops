// Package cameraadapter implements the narrow producer contract used by an
// approved camera/detector service. It deliberately knows nothing about video
// streams or model inference.
package cameraadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Camera struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

type Detection struct {
	EventID    string         `json:"event_id"`
	CameraID   string         `json:"camera_id"`
	Rule       string         `json:"rule"`
	Severity   string         `json:"severity"`
	ObservedAt time.Time      `json:"observed_at"`
	Metadata   map[string]any `json:"metadata"`
}

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, http: httpClient}
}

func (c *Client) DiscoverCamera(ctx context.Context, name string) (Camera, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/ingest/cameras", nil)
	if err != nil {
		return Camera{}, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	response, err := c.http.Do(req)
	if err != nil {
		return Camera{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Camera{}, fmt.Errorf("camera discovery returned %s", response.Status)
	}
	var cameras []Camera
	if err := json.NewDecoder(response.Body).Decode(&cameras); err != nil {
		return Camera{}, err
	}
	for _, camera := range cameras {
		if name == "" || camera.Name == name {
			return camera, nil
		}
	}
	return Camera{}, fmt.Errorf("camera %q was not found for this producer", name)
}

func (c *Client) Heartbeat(ctx context.Context, cameraID string) error {
	return c.post(ctx, "/api/v1/ingest/camera-heartbeats", map[string]string{"camera_id": cameraID}, http.StatusAccepted)
}

func (c *Client) SendDetection(ctx context.Context, detection Detection) error {
	return c.post(ctx, "/api/v1/ingest/detections", detection, http.StatusAccepted, http.StatusOK)
}

func (c *Client) post(ctx context.Context, path string, value any, accepted ...int) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	response, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	for _, status := range accepted {
		if response.StatusCode == status {
			return nil
		}
	}
	return fmt.Errorf("producer request %s returned %s", path, response.Status)
}
