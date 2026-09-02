package visionops

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type App struct {
	DB                         *sql.DB
	Secret, IngestKey          string
	AllowPrivateWebhookTargets bool
	Hub                        *Hub
	Log                        *slog.Logger
	rateMu                     sync.Mutex
	rates                      map[string]rateWindow
	failMu                     sync.Mutex
	failureMode                bool
}
type rateWindow struct {
	started time.Time
	count   int
}
type Hub struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func NewHub() *Hub { return &Hub{clients: map[chan []byte]struct{}{}} }
func (h *Hub) Broadcast(v any) {
	b, _ := json.Marshal(v)
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c <- b:
		default:
		}
	}
}

type Detection struct {
	EventID    string         `json:"event_id"`
	CameraID   string         `json:"camera_id"`
	Rule       string         `json:"rule"`
	Severity   string         `json:"severity"`
	ObservedAt time.Time      `json:"observed_at"`
	Metadata   map[string]any `json:"metadata"`
}

func OpenDB(url string) (*sql.DB, error) {
	config, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(stdlib.GetConnector(*config))
	db.SetMaxOpenConns(12)
	return db, db.Ping()
}
func (a *App) Migrate(ctx context.Context) error {
	if _, err := a.DB.ExecContext(ctx, "create table if not exists schema_migrations (version text primary key, applied_at timestamptz not null default now())"); err != nil {
		return err
	}
	migrationDir := "migrations"
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		migrationDir = "../../migrations"
	}
	files, err := os.ReadDir(migrationDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		var exists bool
		if err := a.DB.QueryRowContext(ctx, "select exists(select 1 from schema_migrations where version=$1)", f.Name()).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		b, err := os.ReadFile(migrationDir + "/" + f.Name())
		if err != nil {
			return err
		}
		tx, err := a.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		for _, s := range strings.Split(string(b), ";") {
			if strings.TrimSpace(s) != "" {
				if _, err = tx.ExecContext(ctx, s); err != nil {
					tx.Rollback()
					return err
				}
			}
		}
		if _, err = tx.ExecContext(ctx, "insert into schema_migrations(version) values($1)", f.Name()); err != nil {
			tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return a.seed(ctx)
}
func (a *App) seed(ctx context.Context) error {
	var n int
	if err := a.DB.QueryRowContext(ctx, "select count(*) from organizations").Scan(&n); err != nil {
		return err
	}
	var org, cam string
	if n == 0 {
		if err := a.DB.QueryRowContext(ctx, "insert into organizations(name) values ('Acme Manufacturing') returning id").Scan(&org); err != nil {
			return err
		}
		if err := a.DB.QueryRowContext(ctx, "insert into cameras(organization_id,name,location) values ($1,'Line A Entrance','Factory floor — Line A') returning id", org).Scan(&cam); err != nil {
			return err
		}
		if _, err := a.DB.ExecContext(ctx, "insert into webhook_subscriptions(organization_id,url,secret) values ($1,'http://localhost:8080/demo/webhook-receiver','demo-webhook-secret')", org); err != nil {
			return err
		}
	} else if err := a.DB.QueryRowContext(ctx, "select id from organizations order by name limit 1").Scan(&org); err != nil {
		return err
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("demo-password"), bcrypt.DefaultCost)
	for email, role := range map[string]string{"admin@acme.test": "admin", "operator@acme.test": "operator", "supervisor@acme.test": "supervisor", "viewer@acme.test": "viewer"} {
		if _, err := a.DB.ExecContext(ctx, "insert into users(organization_id,email,password_hash,role) values($1,$2,$3,$4) on conflict(organization_id,email) do nothing", org, email, string(hash), role); err != nil {
			return err
		}
	}
	key := sha256.Sum256([]byte(a.IngestKey))
	_, err := a.DB.ExecContext(ctx, "insert into api_keys(organization_id,name,key_hash) values($1,'demo simulator',$2) on conflict(key_hash) do nothing", org, fmt.Sprintf("%x", key))
	return err
}
func (a *App) Routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/health", a.health)
	m.HandleFunc("/demo/webhook-receiver", a.demoReceiver)
	m.HandleFunc("/api/v1/demo/failure-mode", a.requireRoles(a.setFailureMode, "admin"))
	m.HandleFunc("/api/v1/auth/login", a.login)
	m.HandleFunc("/api/v1/auth/me", a.auth(a.profile))
	m.HandleFunc("/api/v1/ingest/detections", a.ingest)
	m.HandleFunc("/api/v1/ingest/cameras", a.ingestCameras)
	m.HandleFunc("/api/v1/ingest/camera-heartbeats", a.cameraHeartbeat)
	m.HandleFunc("/api/v1/events", a.auth(a.events))
	m.HandleFunc("/api/v1/incidents", a.auth(a.incidents))
	m.HandleFunc("/api/v1/incidents/", a.auth(a.incidentAction))
	m.HandleFunc("/api/v1/cameras", a.auth(a.cameras))
	m.HandleFunc("/api/v1/webhooks", a.requireRoles(a.webhooks, "admin"))
	m.HandleFunc("/api/v1/users", a.requireRoles(a.users, "admin"))
	m.HandleFunc("/api/v1/api-keys", a.requireRoles(a.apiKeys, "admin"))
	m.HandleFunc("/api/v1/jobs", a.auth(a.jobs))
	m.HandleFunc("/api/v1/jobs/", a.requireRoles(a.jobAction, "admin", "operator"))
	m.HandleFunc("/api/v1/deliveries", a.auth(a.deliveries))
	m.HandleFunc("/api/v1/metrics/operations", a.auth(a.operationsMetrics))
	m.HandleFunc("/api/v1/metrics/observability", a.auth(a.observabilityMetrics))
	m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "web/index.html")
	})
	m.Handle("/", http.FileServer(http.Dir("web")))
	return a.logging(m)
}
func (a *App) health(w http.ResponseWriter, r *http.Request) {
	if err := a.DB.PingContext(r.Context()); err != nil {
		jsonOut(w, 503, map[string]any{"status": "degraded"})
		return
	}
	jsonOut(w, 200, map[string]string{"status": "ok", "service": "visionops"})
}
func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rateKey := "login:" + clientAddress(r)
	if !a.allowRate(rateKey, 10) {
		jsonOut(w, http.StatusTooManyRequests, map[string]string{"error": "too many sign-in attempts"})
		return
	}
	var v struct{ Email, Password string }
	if json.NewDecoder(r.Body).Decode(&v) != nil || strings.TrimSpace(v.Email) == "" || v.Password == "" {
		jsonOut(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		return
	}
	var id, org, hash, role string
	if err := a.DB.QueryRowContext(r.Context(), "select id,organization_id,password_hash,role from users where email=$1 order by created_at limit 1", strings.ToLower(strings.TrimSpace(v.Email))).Scan(&id, &org, &hash, &role); err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(v.Password)) != nil {
		jsonOut(w, 401, map[string]string{"error": "invalid credentials"})
		return
	}
	// A successful authentication clears the attempt window. This retains a
	// per-IP brute-force guard while allowing a shared NAT/proxy to serve many
	// legitimate users without eventually locking all of them out.
	a.clearRate(rateKey)
	jsonOut(w, 200, map[string]string{"token": a.sign(Claims{UserID: id, OrganizationID: org, Role: role, ExpiresAt: time.Now().Add(8 * time.Hour).Unix()})})
}

func (a *App) profile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	claims := r.Context().Value(claimKey{}).(Claims)
	var email, role, organization string
	err := a.DB.QueryRowContext(r.Context(), `select u.email,u.role,o.name from users u join organizations o on o.id=u.organization_id where u.id=$1 and u.organization_id=$2`, claims.UserID, claims.OrganizationID).Scan(&email, &role, &organization)
	if errors.Is(err, sql.ErrNoRows) {
		jsonOut(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if err != nil {
		jsonOut(w, http.StatusInternalServerError, map[string]string{"error": "profile unavailable"})
		return
	}
	jsonOut(w, http.StatusOK, map[string]string{"id": claims.UserID, "email": email, "role": role, "organization_id": claims.OrganizationID, "organization": organization})
}

type Claims struct {
	UserID         string `json:"sub"`
	OrganizationID string `json:"org"`
	Role           string `json:"role"`
	ExpiresAt      int64  `json:"exp"`
}
type claimKey struct{}

func (a *App) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if p == "" {
			p = r.URL.Query().Get("token")
		}
		claims, ok := a.valid(p)
		if !ok {
			jsonOut(w, 401, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), claimKey{}, claims)))
	}
}
func (a *App) requireRoles(next http.HandlerFunc, roles ...string) http.HandlerFunc {
	return a.auth(func(w http.ResponseWriter, r *http.Request) {
		c := r.Context().Value(claimKey{}).(Claims)
		for _, role := range roles {
			if c.Role == role {
				next(w, r)
				return
			}
		}
		jsonOut(w, 403, map[string]string{"error": "forbidden"})
	})
}
func (a *App) sign(claims Claims) string {
	b, _ := json.Marshal(claims)
	p := base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, []byte(a.Secret))
	mac.Write([]byte(p))
	return p + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (a *App) valid(t string) (Claims, bool) {
	x := strings.Split(t, ".")
	if len(x) != 2 {
		return Claims{}, false
	}
	m := hmac.New(sha256.New, []byte(a.Secret))
	m.Write([]byte(x[0]))
	s, _ := base64.RawURLEncoding.DecodeString(x[1])
	if !hmac.Equal(s, m.Sum(nil)) {
		return Claims{}, false
	}
	p, err := base64.RawURLEncoding.DecodeString(x[0])
	var c Claims
	if err != nil || json.Unmarshal(p, &c) != nil || c.UserID == "" || c.OrganizationID == "" || time.Now().Unix() >= c.ExpiresAt {
		return Claims{}, false
	}
	return c, true
}
func orgID(ctx context.Context, db *sql.DB) (string, error) {
	var id string
	err := db.QueryRowContext(ctx, "select id from organizations limit 1").Scan(&id)
	return id, err
}
func currentOrg(ctx context.Context) string { return ctx.Value(claimKey{}).(Claims).OrganizationID }
func (a *App) ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	rawKey := r.Header.Get("X-API-Key")
	keyHash := sha256.Sum256([]byte(rawKey))
	var org string
	if err := a.DB.QueryRowContext(r.Context(), "select organization_id from api_keys where key_hash=$1 and active=true", fmt.Sprintf("%x", keyHash)).Scan(&org); err != nil {
		jsonOut(w, 401, map[string]string{"error": "invalid api key"})
		return
	}
	if !a.allowRate("ingest:"+fmt.Sprintf("%x", keyHash), 60) {
		jsonOut(w, 429, map[string]string{"error": "ingest rate limit exceeded"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var d Detection
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil || d.EventID == "" || d.CameraID == "" || d.Rule == "" {
		jsonOut(w, 400, map[string]string{"error": "event_id, camera_id and rule are required"})
		return
	}
	if d.ObservedAt.IsZero() {
		d.ObservedAt = time.Now().UTC()
	}
	if d.Severity == "" {
		d.Severity = "high"
	}
	var cameraExists bool
	if err := a.DB.QueryRowContext(r.Context(), "select exists(select 1 from cameras where id=$1 and organization_id=$2)", d.CameraID, org).Scan(&cameraExists); err != nil || !cameraExists {
		jsonOut(w, 400, map[string]string{"error": "camera is not registered for this organization"})
		return
	}
	payload, _ := json.Marshal(d.Metadata)
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	var incident string
	err = tx.QueryRowContext(r.Context(), "insert into detection_events(organization_id,producer_event_id,camera_id,rule,severity,observed_at,payload) values($1,$2,$3,$4,$5,$6,$7) returning id", org, d.EventID, d.CameraID, d.Rule, d.Severity, d.ObservedAt, payload).Scan(new(string))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			jsonOut(w, 200, map[string]string{"status": "duplicate"})
			return
		}
		jsonOut(w, 400, map[string]string{"error": "invalid camera or event"})
		return
	}
	err = tx.QueryRowContext(r.Context(), "select id from incidents where organization_id=$1 and camera_id=$2 and rule=$3 and status in ('open','acknowledged') and last_seen_at > $4 order by last_seen_at desc limit 1", org, d.CameraID, d.Rule, d.ObservedAt.Add(-5*time.Minute)).Scan(&incident)
	if errors.Is(err, sql.ErrNoRows) {
		title := strings.ReplaceAll(d.Rule, "_", " ")
		err = tx.QueryRowContext(r.Context(), "insert into incidents(organization_id,camera_id,rule,severity,title,first_seen_at,last_seen_at) values($1,$2,$3,$4,$5,$6,$6) returning id", org, d.CameraID, d.Rule, d.Severity, title, d.ObservedAt).Scan(&incident)
	} else if err == nil {
		_, err = tx.ExecContext(r.Context(), "update incidents set last_seen_at=$1, occurrences=occurrences+1, updated_at=now() where id=$2", d.ObservedAt, incident)
	}
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	_, err = tx.ExecContext(r.Context(), "insert into incident_activity(incident_id,type,detail) values($1,'detection_received',$2::jsonb)", incident, payload)
	if err == nil {
		_, err = tx.ExecContext(r.Context(), "insert into outbox_jobs(organization_id,topic,payload) values($1,'incident.created',jsonb_build_object('incident_id',$2::uuid))", org, incident)
	}
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if err = tx.Commit(); err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	a.Hub.Broadcast(map[string]string{"type": "incident.updated", "incident_id": incident})
	jsonOut(w, 202, map[string]string{"status": "accepted", "incident_id": incident})
}
func (a *App) allowRate(key string, limit int) bool {
	a.rateMu.Lock()
	defer a.rateMu.Unlock()
	if a.rates == nil {
		a.rates = map[string]rateWindow{}
	}
	now := time.Now()
	for existingKey, existing := range a.rates {
		if now.Sub(existing.started) >= time.Minute {
			delete(a.rates, existingKey)
		}
	}
	v := a.rates[key]
	if v.started.IsZero() || now.Sub(v.started) >= time.Minute {
		a.rates[key] = rateWindow{started: now, count: 1}
		return true
	}
	if v.count >= limit {
		return false
	}
	v.count++
	a.rates[key] = v
	return true
}

func (a *App) clearRate(key string) {
	a.rateMu.Lock()
	defer a.rateMu.Unlock()
	delete(a.rates, key)
}

func clientAddress(r *http.Request) string {
	if ip := net.ParseIP(r.Header.Get("CF-Connecting-IP")); ip != nil {
		return ip.String()
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
func (a *App) incidents(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	rows, err := a.DB.QueryContext(r.Context(), "select i.id,i.title,i.rule,i.severity,i.status,i.occurrences,i.first_seen_at,i.last_seen_at,c.name,c.location from incidents i join cameras c on c.id=i.camera_id where i.organization_id=$1 order by i.last_seen_at desc limit $2", currentOrg(r.Context()), limit)
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, title, rule, sev, status, name, loc string
		var n int
		var first, last time.Time
		rows.Scan(&id, &title, &rule, &sev, &status, &n, &first, &last, &name, &loc)
		out = append(out, map[string]any{"id": id, "title": title, "rule": rule, "severity": sev, "status": status, "occurrences": n, "first_seen_at": first, "last_seen_at": last, "camera": map[string]string{"name": name, "location": loc}})
	}
	jsonOut(w, 200, out)
}
func (a *App) incidentAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/incidents/"), "/")
	if r.Method == "GET" && len(parts) == 1 {
		a.incidentDetail(w, r, parts[0])
		return
	}
	if len(parts) != 2 || r.Method != "POST" {
		w.WriteHeader(404)
		return
	}
	claims := r.Context().Value(claimKey{}).(Claims)
	if claims.Role != "admin" && claims.Role != "operator" {
		jsonOut(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id, action := parts[0], parts[1]
	var input struct {
		Note string `json:"note"`
	}
	json.NewDecoder(r.Body).Decode(&input)
	q := ""
	typ := ""
	switch action {
	case "acknowledge":
		q = "update incidents set status='acknowledged',acknowledged_at=now(),updated_at=now() where id=$1 and organization_id=$2 and status='open'"
		typ = "acknowledged"
	case "resolve":
		q = "update incidents set status='resolved',resolved_at=now(),resolution_note=$3,updated_at=now() where id=$1 and organization_id=$2 and status!='resolved'"
		typ = "resolved"
	default:
		w.WriteHeader(404)
		return
	}
	args := []any{id, currentOrg(r.Context())}
	if action == "resolve" {
		args = append(args, input.Note)
	}
	res, err := a.DB.ExecContext(r.Context(), q, args...)
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		jsonOut(w, 409, map[string]string{"error": "invalid state or incident"})
		return
	}
	a.DB.ExecContext(r.Context(), "insert into incident_activity(incident_id,type,actor_user_id,note) values($1,$2,$3,$4)", id, typ, claims.UserID, input.Note)
	a.Hub.Broadcast(map[string]string{"type": "incident." + typ, "incident_id": id})
	jsonOut(w, 200, map[string]string{"status": typ})
}
func (a *App) incidentDetail(w http.ResponseWriter, r *http.Request, id string) {
	org := currentOrg(r.Context())
	var title, rule, sev, status, note, name, loc string
	var occurrences int
	var first, last time.Time
	var resolved sql.NullTime
	err := a.DB.QueryRowContext(r.Context(), "select i.title,i.rule,i.severity,i.status,coalesce(i.resolution_note,''),i.occurrences,i.first_seen_at,i.last_seen_at,i.resolved_at,c.name,c.location from incidents i join cameras c on c.id=i.camera_id where i.id=$1 and i.organization_id=$2", id, org).Scan(&title, &rule, &sev, &status, &note, &occurrences, &first, &last, &resolved, &name, &loc)
	if err == sql.ErrNoRows {
		jsonOut(w, 404, map[string]string{"error": "incident not found"})
		return
	}
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), "select a.type,coalesce(a.note,''),a.created_at,coalesce(u.email,'system') from incident_activity a left join users u on u.id=a.actor_user_id where a.incident_id=$1 order by a.created_at", id)
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	activity := []map[string]any{}
	for rows.Next() {
		var typ, n, actor string
		var at time.Time
		rows.Scan(&typ, &n, &at, &actor)
		activity = append(activity, map[string]any{"type": typ, "note": n, "actor": actor, "created_at": at})
	}
	jsonOut(w, 200, map[string]any{"id": id, "title": title, "rule": rule, "severity": sev, "status": status, "resolution_note": note, "occurrences": occurrences, "first_seen_at": first, "last_seen_at": last, "resolved_at": resolved, "camera": map[string]string{"name": name, "location": loc}, "activity": activity})
}
func (a *App) cameras(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		c := r.Context().Value(claimKey{}).(Claims)
		if c.Role != "admin" {
			jsonOut(w, 403, map[string]string{"error": "forbidden"})
			return
		}
		var v struct{ Name, Location string }
		json.NewDecoder(r.Body).Decode(&v)
		if strings.TrimSpace(v.Name) == "" || strings.TrimSpace(v.Location) == "" {
			jsonOut(w, 400, map[string]string{"error": "name and location are required"})
			return
		}
		_, err := a.DB.ExecContext(r.Context(), "insert into cameras(organization_id,name,location) values($1,$2,$3)", currentOrg(r.Context()), v.Name, v.Location)
		if err != nil {
			jsonOut(w, 500, map[string]string{"error": err.Error()})
			return
		}
		jsonOut(w, 201, map[string]string{"status": "created"})
		return
	}
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	rows, _ := a.DB.QueryContext(r.Context(), "select c.id,c.name,c.location,case when c.last_heartbeat_at is null then 'offline' when c.last_heartbeat_at < now()-interval '5 minutes' then 'degraded' else 'online' end,coalesce(max(d.observed_at)::text,'never') from cameras c left join detection_events d on d.camera_id=c.id and d.organization_id=c.organization_id where c.organization_id=$1 group by c.id order by c.name", currentOrg(r.Context()))
	defer rows.Close()
	out := []map[string]string{}
	for rows.Next() {
		var i, n, l, s, last string
		rows.Scan(&i, &n, &l, &s, &last)
		out = append(out, map[string]string{"id": i, "name": n, "location": l, "status": s, "last_detection": last})
	}
	jsonOut(w, 200, out)
}
func (a *App) cameraHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	sum := sha256.Sum256([]byte(r.Header.Get("X-API-Key")))
	var org string
	if err := a.DB.QueryRowContext(r.Context(), "select organization_id from api_keys where key_hash=$1 and active", fmt.Sprintf("%x", sum)).Scan(&org); err != nil {
		jsonOut(w, 401, map[string]string{"error": "invalid api key"})
		return
	}
	var v struct {
		CameraID string `json:"camera_id"`
	}
	if json.NewDecoder(r.Body).Decode(&v) != nil || v.CameraID == "" {
		jsonOut(w, 400, map[string]string{"error": "camera_id is required"})
		return
	}
	res, err := a.DB.ExecContext(r.Context(), "update cameras set last_heartbeat_at=now() where id=$1 and organization_id=$2", v.CameraID, org)
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		jsonOut(w, 404, map[string]string{"error": "camera not found"})
		return
	}
	jsonOut(w, 202, map[string]string{"status": "accepted"})
}

func (a *App) ingestCameras(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	sum := sha256.Sum256([]byte(r.Header.Get("X-API-Key")))
	var org string
	if err := a.DB.QueryRowContext(r.Context(), "select organization_id from api_keys where key_hash=$1 and active", fmt.Sprintf("%x", sum)).Scan(&org); err != nil {
		jsonOut(w, http.StatusUnauthorized, map[string]string{"error": "invalid api key"})
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), "select id,name,location from cameras where organization_id=$1 order by name", org)
	if err != nil {
		jsonOut(w, http.StatusInternalServerError, map[string]string{"error": "camera discovery unavailable"})
		return
	}
	defer rows.Close()
	out := []map[string]string{}
	for rows.Next() {
		var id, name, location string
		if err := rows.Scan(&id, &name, &location); err != nil {
			jsonOut(w, http.StatusInternalServerError, map[string]string{"error": "camera discovery unavailable"})
			return
		}
		out = append(out, map[string]string{"id": id, "name": name, "location": location})
	}
	jsonOut(w, http.StatusOK, out)
}
func (a *App) users(w http.ResponseWriter, r *http.Request) {
	org := currentOrg(r.Context())
	if r.Method == "GET" {
		rows, err := a.DB.QueryContext(r.Context(), "select id,email,role,created_at from users where organization_id=$1 order by created_at", org)
		if err != nil {
			jsonOut(w, 500, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, email, role string
			var at time.Time
			rows.Scan(&id, &email, &role, &at)
			out = append(out, map[string]any{"id": id, "email": email, "role": role, "created_at": at})
		}
		jsonOut(w, 200, out)
		return
	}
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	var v struct{ Email, Password, Role string }
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil || !strings.Contains(v.Email, "@") || len(v.Password) < 12 || !validRole(v.Role) {
		jsonOut(w, 400, map[string]string{"error": "email, 12-character password, and valid role are required"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(v.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	_, err = a.DB.ExecContext(r.Context(), "insert into users(organization_id,email,password_hash,role) values($1,$2,$3,$4)", org, strings.ToLower(v.Email), string(hash), v.Role)
	if err != nil {
		jsonOut(w, 409, map[string]string{"error": "user already exists"})
		return
	}
	jsonOut(w, 201, map[string]string{"status": "created"})
}
func validRole(role string) bool {
	return role == "admin" || role == "operator" || role == "supervisor" || role == "viewer"
}
func (a *App) apiKeys(w http.ResponseWriter, r *http.Request) {
	org := currentOrg(r.Context())
	if r.Method == "GET" {
		rows, err := a.DB.QueryContext(r.Context(), "select id,name,active,created_at from api_keys where organization_id=$1 order by created_at desc", org)
		if err != nil {
			jsonOut(w, 500, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, name string
			var active bool
			var at time.Time
			rows.Scan(&id, &name, &active, &at)
			out = append(out, map[string]any{"id": id, "name": name, "active": active, "created_at": at})
		}
		jsonOut(w, 200, out)
		return
	}
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	var v struct{ Name string }
	json.NewDecoder(r.Body).Decode(&v)
	if strings.TrimSpace(v.Name) == "" {
		jsonOut(w, 400, map[string]string{"error": "name is required"})
		return
	}
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	key := "vo_" + base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(key))
	_, err := a.DB.ExecContext(r.Context(), "insert into api_keys(organization_id,name,key_hash) values($1,$2,$3)", org, v.Name, fmt.Sprintf("%x", sum))
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	jsonOut(w, 201, map[string]string{"api_key": key, "warning": "Store this key now; it cannot be retrieved again."})
}
func (a *App) webhooks(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, _ := a.DB.QueryContext(r.Context(), "select id,url,enabled,created_at from webhook_subscriptions where organization_id=$1 order by created_at desc", currentOrg(r.Context()))
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, u string
			var e bool
			var c time.Time
			rows.Scan(&id, &u, &e, &c)
			out = append(out, map[string]any{"id": id, "url": u, "enabled": e, "created_at": c})
		}
		jsonOut(w, 200, out)
		return
	}
	var x struct{ URL, Secret string }
	if err := json.NewDecoder(r.Body).Decode(&x); err != nil || len(x.Secret) < 12 {
		jsonOut(w, 400, map[string]string{"error": "valid HTTPS URL and 12-character signing secret are required"})
		return
	}
	if err := validateWebhookURL(x.URL); err != nil {
		jsonOut(w, 400, map[string]string{"error": err.Error()})
		return
	}
	org := currentOrg(r.Context())
	_, err := a.DB.ExecContext(r.Context(), "insert into webhook_subscriptions(organization_id,url,secret) values($1,$2,$3)", org, x.URL, x.Secret)
	if err != nil {
		jsonOut(w, 400, map[string]string{"error": err.Error()})
		return
	}
	jsonOut(w, 201, map[string]string{"status": "created"})
}
func (a *App) jobs(w http.ResponseWriter, r *http.Request) {
	rows, _ := a.DB.QueryContext(r.Context(), "select id,topic,status,attempts,available_at,last_error from outbox_jobs where organization_id=$1 order by created_at desc limit 100", currentOrg(r.Context()))
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, t, s string
		var a int
		var at time.Time
		var e sql.NullString
		rows.Scan(&id, &t, &s, &a, &at, &e)
		out = append(out, map[string]any{"id": id, "topic": t, "status": s, "attempts": a, "available_at": at, "last_error": e.String})
	}
	jsonOut(w, 200, out)
}
func (a *App) jobAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/"), "/")
	if r.Method != "POST" || len(parts) != 2 || parts[1] != "replay" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	res, err := a.DB.ExecContext(r.Context(), "update outbox_jobs set status='pending',attempts=0,available_at=now(),last_error=null,completed_at=null where id=$1 and organization_id=$2 and status='dead'", parts[0], currentOrg(r.Context()))
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		jsonOut(w, 409, map[string]string{"error": "only dead jobs can be replayed"})
		return
	}
	a.Hub.Broadcast(map[string]string{"type": "job.replayed", "job_id": parts[0]})
	jsonOut(w, 200, map[string]string{"status": "queued"})
}
func (a *App) deliveries(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.QueryContext(r.Context(), "select d.id,d.job_id,d.subscription_id,d.status_code,d.error,d.attempted_at from webhook_deliveries d join outbox_jobs j on j.id=d.job_id where j.organization_id=$1 order by d.attempted_at desc limit 100", currentOrg(r.Context()))
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, job, sub string
		var code sql.NullInt64
		var msg sql.NullString
		var at time.Time
		rows.Scan(&id, &job, &sub, &code, &msg, &at)
		out = append(out, map[string]any{"id": id, "job_id": job, "subscription_id": sub, "status_code": code.Int64, "error": msg.String, "attempted_at": at})
	}
	jsonOut(w, 200, out)
}
func (a *App) operationsMetrics(w http.ResponseWriter, r *http.Request) {
	org := currentOrg(r.Context())
	out := map[string]any{}
	rows, err := a.DB.QueryContext(r.Context(), "select severity,count(*) from incidents where organization_id=$1 and status!='resolved' group by severity order by severity", org)
	if err != nil {
		jsonOut(w, 500, map[string]string{"error": err.Error()})
		return
	}
	sev := map[string]int{}
	for rows.Next() {
		var s string
		var n int
		rows.Scan(&s, &n)
		sev[s] = n
	}
	rows.Close()
	out["open_by_severity"] = sev
	rows, _ = a.DB.QueryContext(r.Context(), "select rule,count(*) from incidents where organization_id=$1 group by rule order by count(*) desc limit 5", org)
	rules := []map[string]any{}
	for rows.Next() {
		var rule string
		var n int
		rows.Scan(&rule, &n)
		rules = append(rules, map[string]any{"rule": rule, "count": n})
	}
	rows.Close()
	var ack, res sql.NullFloat64
	a.DB.QueryRowContext(r.Context(), "select avg(extract(epoch from acknowledged_at-first_seen_at)/60),avg(extract(epoch from resolved_at-first_seen_at)/60) from incidents where organization_id=$1", org).Scan(&ack, &res)
	out["average_ack_minutes"] = ack.Float64
	out["average_resolution_minutes"] = res.Float64
	out["recurring_rules"] = rules
	jsonOut(w, 200, out)
}
func (a *App) events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	f, ok := w.(http.Flusher)
	if !ok {
		return
	}
	ch := make(chan []byte, 8)
	a.Hub.mu.Lock()
	a.Hub.clients[ch] = struct{}{}
	a.Hub.mu.Unlock()
	defer func() { a.Hub.mu.Lock(); delete(a.Hub.clients, ch); a.Hub.mu.Unlock() }()
	fmt.Fprint(w, "event: connected\ndata: {}\n\n")
	f.Flush()
	for {
		select {
		case b := <-ch:
			fmt.Fprintf(w, "event: update\ndata: %s\n\n", b)
			f.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
func (a *App) demoReceiver(w http.ResponseWriter, r *http.Request) {
	a.failMu.Lock()
	fail := a.failureMode
	a.failMu.Unlock()
	if fail {
		jsonOut(w, 500, map[string]string{"error": "simulated webhook failure"})
		return
	}
	b, _ := io.ReadAll(r.Body)
	a.Log.Info("demo webhook received", "payload", string(b))
	jsonOut(w, 200, map[string]string{"status": "received"})
}
func (a *App) setFailureMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	var v struct {
		Enabled bool `json:"enabled"`
	}
	json.NewDecoder(r.Body).Decode(&v)
	a.failMu.Lock()
	a.failureMode = v.Enabled
	a.failMu.Unlock()
	jsonOut(w, 200, map[string]bool{"enabled": v.Enabled})
}
func (a *App) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'; object-src 'none'; connect-src 'self'; media-src 'self'")
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		start := time.Now()
		next.ServeHTTP(w, r)
		a.Log.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String())
	})
}

func validateWebhookURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return errors.New("webhook URL must be an absolute HTTPS URL without credentials")
	}
	ips, err := net.LookupIP(u.Hostname())
	if err != nil || len(ips) == 0 {
		return errors.New("webhook host could not be resolved")
	}
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return errors.New("webhook URL must not target a private or local address")
		}
	}
	return nil
}

func isPublicIP(ip net.IP) bool {
	return !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsUnspecified() && !ip.IsMulticast()
}
func jsonOut(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (a *App) observabilityMetrics(w http.ResponseWriter, r *http.Request) {
	org := currentOrg(r.Context())
	q := func(sql string) int { var n int; a.DB.QueryRowContext(r.Context(), sql, org).Scan(&n); return n }
	jsonOut(w, 200, map[string]int{"detections": q("select count(*) from detection_events where organization_id=$1"), "incidents": q("select count(*) from incidents where organization_id=$1"), "outbox_pending": q("select count(*) from outbox_jobs where organization_id=$1 and status='pending'"), "outbox_dead": q("select count(*) from outbox_jobs where organization_id=$1 and status='dead'"), "webhook_deliveries": q("select count(*) from webhook_deliveries d join outbox_jobs j on j.id=d.job_id where j.organization_id=$1"), "webhook_failures": q("select count(*) from webhook_deliveries d join outbox_jobs j on j.id=d.job_id where j.organization_id=$1 and (d.status_code is null or d.status_code<200 or d.status_code>=300)")})
}
