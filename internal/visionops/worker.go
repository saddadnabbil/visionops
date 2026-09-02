package visionops

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func (a *App) StartWorker(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			a.workOne(ctx)
		}
	}
}
func (a *App) workOne(ctx context.Context) {
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	var id, org string
	var payload []byte
	var attempts int
	err = tx.QueryRowContext(ctx, "select id,organization_id,payload,attempts from outbox_jobs where status='pending' and available_at<=now() order by created_at for update skip locked limit 1").Scan(&id, &org, &payload, &attempts)
	if err != nil {
		return
	}
	tx.ExecContext(ctx, "update outbox_jobs set status='processing',attempts=attempts+1 where id=$1", id)
	if tx.Commit() != nil {
		return
	}
	err = a.deliver(ctx, id, org, payload)
	if err == nil {
		a.DB.ExecContext(ctx, "update outbox_jobs set status='done',completed_at=now() where id=$1", id)
		return
	}
	next := time.Now().Add(time.Duration(1<<min(attempts, 5)) * time.Second)
	status := "pending"
	if attempts+1 >= 5 {
		status = "dead"
	}
	a.DB.ExecContext(ctx, "update outbox_jobs set status=$1,available_at=$2,last_error=$3 where id=$4", status, next, err.Error(), id)
	a.Hub.Broadcast(map[string]string{"type": "job." + status, "job_id": id})
}
func (a *App) deliver(ctx context.Context, job, org string, payload []byte) error {
	rows, err := a.DB.QueryContext(ctx, "select id,url,secret from webhook_subscriptions where organization_id=$1 and enabled", org)
	if err != nil {
		return err
	}
	defer rows.Close()
	had := false
	for rows.Next() {
		had = true
		var id, url, secret string
		rows.Scan(&id, &url, &secret)
		req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(payload)))
		if err != nil {
			return err
		}
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-VisionOps-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		req.Header.Set("X-VisionOps-Timestamp", fmt.Sprint(time.Now().Unix()))
		client := safeWebhookClient
		if a.AllowPrivateWebhookTargets {
			client = &http.Client{Timeout: 4 * time.Second}
		}
		res, err := client.Do(req)
		code := 0
		msg := ""
		if err == nil {
			code = res.StatusCode
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			msg = string(body)
		} else {
			msg = err.Error()
		}
		a.DB.ExecContext(ctx, "insert into webhook_deliveries(job_id,subscription_id,status_code,error) values($1,$2,$3,$4)", job, id, code, msg)
		if err != nil || code < 200 || code >= 300 {
			return fmt.Errorf("delivery to %s failed: %s", url, msg)
		}
	}
	if !had {
		return nil
	}
	return rows.Err()
}

var safeWebhookClient = &http.Client{Timeout: 4 * time.Second, Transport: &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("webhook host could not be resolved")
	}
	for _, ip := range ips {
		if !isPublicIP(net.IP(ip.AsSlice())) {
			return nil, fmt.Errorf("webhook host resolved to a private or local address")
		}
	}
	return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, network, address)
}}}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var _ = json.RawMessage{}
var _ = sql.ErrNoRows
