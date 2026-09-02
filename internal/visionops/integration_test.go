//go:build integration

package visionops

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func integrationApp(t *testing.T) *App {
	t.Helper()
	url := os.Getenv("VISIONOPS_INTEGRATION_DATABASE_URL")
	if url == "" {
		url = "postgres://visionops:visionops@localhost:5433/visionops?sslmode=disable"
	}
	db, err := OpenDB(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	a := &App{DB: db, Secret: "integration-secret", IngestKey: "unused", Hub: NewHub(), Log: slog.New(slog.NewTextHandler(os.Stdout, nil))}
	if err := a.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return a
}
func testOrg(t *testing.T, a *App, name string) (org, camera, key string) {
	t.Helper()
	ctx := context.Background()
	if err := a.DB.QueryRowContext(ctx, "insert into organizations(name) values($1) returning id", name).Scan(&org); err != nil {
		t.Fatal(err)
	}
	if err := a.DB.QueryRowContext(ctx, "insert into cameras(organization_id,name,location) values($1,'test camera','test') returning id", org).Scan(&camera); err != nil {
		t.Fatal(err)
	}
	key = "vo_test_" + name
	sum := sha256.Sum256([]byte(key))
	if _, err := a.DB.ExecContext(ctx, "insert into api_keys(organization_id,name,key_hash) values($1,'test',$2)", org, fmt.Sprintf("%x", sum)); err != nil {
		t.Fatal(err)
	}
	return
}
func ingestRequest(t *testing.T, a *App, key, camera, event string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(Detection{EventID: event, CameraID: camera, Rule: "missing_ppe", Severity: "high", ObservedAt: time.Now().UTC(), Metadata: map[string]any{"test": true}})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/detections", bytes.NewReader(body))
	r.Header.Set("X-API-Key", key)
	w := httptest.NewRecorder()
	a.ingest(w, r)
	return w
}

func TestIntegrationTenantIsolationAndOutboxDelivery(t *testing.T) {
	a := integrationApp(t)
	suffix := fmt.Sprint(time.Now().UnixNano())
	orgA, camA, keyA := testOrg(t, a, "tenant-a-"+suffix)
	_, camB, _ := testOrg(t, a, "tenant-b-"+suffix)
	if got := ingestRequest(t, a, keyA, camB, "cross-"+suffix).Code; got != http.StatusBadRequest {
		t.Fatalf("cross-tenant camera accepted: %d", got)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	defer server.Close()
	if _, err := a.DB.Exec("insert into webhook_subscriptions(organization_id,url,secret) values($1,$2,'test-secret')", orgA, server.URL); err != nil {
		t.Fatal(err)
	}
	if got := ingestRequest(t, a, keyA, camA, "valid-"+suffix).Code; got != http.StatusAccepted {
		t.Fatalf("valid detection status: %d", got)
	}
	for i := 0; i < 5; i++ {
		a.workOne(context.Background())
		var pending int
		a.DB.QueryRow("select count(*) from outbox_jobs where organization_id=$1 and status='pending'", orgA).Scan(&pending)
		if pending == 0 {
			break
		}
	}
	var incidents, done, deliveries int
	a.DB.QueryRow("select count(*) from incidents where organization_id=$1", orgA).Scan(&incidents)
	a.DB.QueryRow("select count(*) from outbox_jobs where organization_id=$1 and status='done'", orgA).Scan(&done)
	a.DB.QueryRow("select count(*) from webhook_deliveries d join outbox_jobs j on j.id=d.job_id where j.organization_id=$1", orgA).Scan(&deliveries)
	if incidents != 1 || done != 1 || deliveries != 1 {
		t.Fatalf("want incident/job/delivery 1/1/1, got %d/%d/%d", incidents, done, deliveries)
	}
}

func TestIntegrationFailedDeliveryIsRetainedForRetry(t *testing.T) {
	a := integrationApp(t)
	suffix := fmt.Sprint(time.Now().UnixNano())
	org, _, _ := testOrg(t, a, "retry-"+suffix)
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) }))
	defer failing.Close()
	if _, err := a.DB.Exec("insert into webhook_subscriptions(organization_id,url,secret) values($1,$2,'test-secret')", org, failing.URL); err != nil {
		t.Fatal(err)
	}
	var job string
	if err := a.DB.QueryRow("insert into outbox_jobs(organization_id,topic,payload) values($1,'incident.created','{}') returning id", org).Scan(&job); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		a.workOne(context.Background())
		var attempts int
		a.DB.QueryRow("select attempts from outbox_jobs where id=$1", job).Scan(&attempts)
		if attempts > 0 {
			break
		}
	}
	var status string
	var attempts, deliveries int
	a.DB.QueryRow("select status,attempts from outbox_jobs where id=$1", job).Scan(&status, &attempts)
	a.DB.QueryRow("select count(*) from webhook_deliveries where job_id=$1", job).Scan(&deliveries)
	if status != "pending" || attempts != 1 || deliveries != 1 {
		t.Fatalf("want retained pending job with one attempt/delivery, got %s/%d/%d", status, attempts, deliveries)
	}
}

func TestIntegrationHumanRolesCanReadIncidentDetailButOnlyRespondersCanMutate(t *testing.T) {
	a := integrationApp(t)
	suffix := fmt.Sprint(time.Now().UnixNano())
	org, camera, key := testOrg(t, a, "roles-"+suffix)
	if got := ingestRequest(t, a, key, camera, "role-event-"+suffix).Code; got != http.StatusAccepted {
		t.Fatalf("ingest status: %d", got)
	}
	var incident string
	if err := a.DB.QueryRow("select id from incidents where organization_id=$1 order by created_at desc limit 1", org).Scan(&incident); err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("integration-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"admin", "operator", "supervisor", "viewer"} {
		var user string
		email := role + "-" + suffix + "@example.test"
		if err := a.DB.QueryRow("insert into users(organization_id,email,password_hash,role) values($1,$2,$3,$4) returning id", org, email, string(hash), role).Scan(&user); err != nil {
			t.Fatal(err)
		}
		token := a.sign(Claims{UserID: user, OrganizationID: org, Role: role, ExpiresAt: time.Now().Add(time.Minute).Unix()})
		detail := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/"+incident, nil)
		detail.Header.Set("Authorization", "Bearer "+token)
		detailResult := httptest.NewRecorder()
		a.Routes().ServeHTTP(detailResult, detail)
		if detailResult.Code != http.StatusOK {
			t.Errorf("%s detail status: got %d want 200", role, detailResult.Code)
		}

		ack := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+incident+"/acknowledge", bytes.NewBufferString(`{"note":"claimed"}`))
		ack.Header.Set("Authorization", "Bearer "+token)
		ack.Header.Set("Content-Type", "application/json")
		ackResult := httptest.NewRecorder()
		a.Routes().ServeHTTP(ackResult, ack)
		if role == "admin" || role == "operator" {
			if ackResult.Code != http.StatusOK && ackResult.Code != http.StatusConflict {
				t.Errorf("%s acknowledge status: got %d want 200/409", role, ackResult.Code)
			}
		} else if ackResult.Code != http.StatusForbidden {
			t.Errorf("%s acknowledge status: got %d want 403", role, ackResult.Code)
		}
	}
}

func TestIntegrationLoginReturnsSessionProfile(t *testing.T) {
	a := integrationApp(t)
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"operator@acme.test","password":"demo-password"}`))
	login.Header.Set("Content-Type", "application/json")
	loginResult := httptest.NewRecorder()
	a.Routes().ServeHTTP(loginResult, login)
	if loginResult.Code != http.StatusOK {
		t.Fatalf("login status: got %d body %s", loginResult.Code, loginResult.Body.String())
	}
	var session struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loginResult.Body.Bytes(), &session); err != nil || session.Token == "" {
		t.Fatalf("login did not return token: %v", err)
	}
	profile := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	profile.Header.Set("Authorization", "Bearer "+session.Token)
	profileResult := httptest.NewRecorder()
	a.Routes().ServeHTTP(profileResult, profile)
	if profileResult.Code != http.StatusOK {
		t.Fatalf("profile status: got %d body %s", profileResult.Code, profileResult.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(profileResult.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["email"] != "operator@acme.test" || got["role"] != "operator" || got["organization"] != "Acme Manufacturing" {
		t.Fatalf("unexpected profile: %#v", got)
	}
}

func TestIntegrationProducerCanDiscoverOnlyTenantOwnedCameras(t *testing.T) {
	a := integrationApp(t)
	suffix := fmt.Sprint(time.Now().UnixNano())
	_, cameraA, keyA := testOrg(t, a, "producer-a-"+suffix)
	_, cameraB, _ := testOrg(t, a, "producer-b-"+suffix)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ingest/cameras", nil)
	req.Header.Set("X-API-Key", keyA)
	result := httptest.NewRecorder()
	a.Routes().ServeHTTP(result, req)
	if result.Code != http.StatusOK {
		t.Fatalf("camera discovery status: got %d body %s", result.Code, result.Body.String())
	}
	var cameras []map[string]string
	if err := json.Unmarshal(result.Body.Bytes(), &cameras); err != nil {
		t.Fatal(err)
	}
	if len(cameras) != 1 || cameras[0]["id"] != cameraA || cameras[0]["id"] == cameraB {
		t.Fatalf("unexpected producer camera discovery: %#v", cameras)
	}
}
