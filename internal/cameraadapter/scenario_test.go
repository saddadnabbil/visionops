package cameraadapter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadScenarioRejectsUnknownSafetyRule(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scenario.json")
	if err := os.WriteFile(path, []byte(`[{"after_seconds":1,"rule":"unknown_rule","severity":"high"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadScenario(path); err == nil {
		t.Fatal("unknown safety rule must be rejected before the producer sends it")
	}
}

func TestLoadScenarioAcceptsDisclosedRecordedScenario(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recorded.json")
	content := `{"mode":"recorded_scenario","source_asset":"wide.mp4","disclosure":"RECORDED SCENARIO","events":[{"after_seconds":2,"rule":"missing_ppe","severity":"high"}]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	scenario, err := LoadScenario(path)
	if err != nil {
		t.Fatal(err)
	}
	if scenario.Mode != "recorded_scenario" || scenario.Metadata()["disclosure"] != "RECORDED SCENARIO" {
		t.Fatalf("recorded scenario provenance was lost: %#v", scenario)
	}
}
