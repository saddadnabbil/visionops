package cameraadapter

import (
	"encoding/json"
	"fmt"
	"os"
)

type ScenarioEvent struct {
	AfterSeconds int            `json:"after_seconds"`
	Rule         string         `json:"rule"`
	Severity     string         `json:"severity"`
	Metadata     map[string]any `json:"metadata"`
}

// Scenario is either a legacy event array or a named, disclosed recorded
// scenario. Recorded scenarios deliberately carry no frames, faces, or stream
// URLs; they only provide safe provenance for an already scripted detection.
type Scenario struct {
	Mode        string          `json:"mode"`
	SourceAsset string          `json:"source_asset"`
	SourceURL   string          `json:"source_url"`
	Disclosure  string          `json:"disclosure"`
	Events      []ScenarioEvent `json:"events"`
}

func LoadScenario(path string) (Scenario, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Scenario{}, err
	}
	var scenario Scenario
	if err := json.Unmarshal(content, &scenario); err != nil {
		var legacy []ScenarioEvent
		if legacyErr := json.Unmarshal(content, &legacy); legacyErr != nil {
			return Scenario{}, err
		}
		scenario = Scenario{Mode: "fixture", Events: legacy}
	}
	if len(scenario.Events) == 0 {
		return Scenario{}, fmt.Errorf("camera adapter scenario has no events")
	}
	if scenario.Mode != "fixture" && scenario.Mode != "recorded_scenario" {
		return Scenario{}, fmt.Errorf("camera adapter scenario mode is invalid")
	}
	if scenario.Mode == "recorded_scenario" && (scenario.SourceAsset == "" || scenario.Disclosure == "") {
		return Scenario{}, fmt.Errorf("recorded scenario requires source_asset and disclosure")
	}
	for index, event := range scenario.Events {
		if event.AfterSeconds < 0 || !validRule(event.Rule) || !validSeverity(event.Severity) {
			return Scenario{}, fmt.Errorf("camera adapter scenario event %d is invalid", index+1)
		}
	}
	return scenario, nil
}

func (s Scenario) Metadata() map[string]any {
	if s.Mode != "recorded_scenario" {
		return map[string]any{"source_mode": "fixture"}
	}
	return map[string]any{"source_mode": s.Mode, "source_asset": s.SourceAsset, "source_url": s.SourceURL, "disclosure": s.Disclosure}
}

func validRule(rule string) bool {
	return rule == "missing_ppe" || rule == "restricted_area" || rule == "crowding"
}

func validSeverity(severity string) bool {
	return severity == "low" || severity == "medium" || severity == "high" || severity == "critical"
}
