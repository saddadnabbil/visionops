package visionops

import (
	"testing"
	"time"
)

func TestSignedTokenValidation(t *testing.T) {
	a := &App{Secret: "test-secret"}
	valid := Claims{UserID: "user", OrganizationID: "org", Role: "operator", ExpiresAt: time.Now().Add(time.Minute).Unix()}
	if _, ok := a.valid(a.sign(valid)); !ok {
		t.Fatal("fresh signed token should validate")
	}
	expired := Claims{UserID: "user", OrganizationID: "org", Role: "operator", ExpiresAt: time.Now().Add(-time.Minute).Unix()}
	if _, ok := a.valid(a.sign(expired)); ok {
		t.Fatal("expired token must not validate")
	}
	if _, ok := a.valid("tampered.token"); ok {
		t.Fatal("tampered token must not validate")
	}
}

func TestHubBroadcast(t *testing.T) {
	h := NewHub()
	c := make(chan []byte, 1)
	h.clients[c] = struct{}{}
	h.Broadcast(map[string]string{"type": "incident.updated"})
	if got := string(<-c); got != `{"type":"incident.updated"}` {
		t.Fatalf("unexpected event: %s", got)
	}
}
