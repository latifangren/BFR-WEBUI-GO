package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/handlers"
	"bfr-webui-go/internal/power"
)

func TestHandlePowerSchedule_GetAndPost(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// 1. GET Schedule Status
	reqGet := httptest.NewRequest(http.MethodGet, "/api/power/schedule", nil)
	rrGet := httptest.NewRecorder()
	handlers.HandlePowerSchedule(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Fatalf("expected status 200 for GET, got %d", rrGet.Code)
	}

	var status power.ScheduleStatus
	if err := json.NewDecoder(rrGet.Body).Decode(&status); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// 2. POST Valid Interval Config
	postCfg := power.ScheduleConfig{
		Enabled:       true,
		Mode:          power.ScheduleModeInterval,
		IntervalHours: 6,
	}
	body, _ := json.Marshal(postCfg)
	reqPost := httptest.NewRequest(http.MethodPost, "/api/power/schedule", bytes.NewReader(body))
	rrPost := httptest.NewRecorder()
	handlers.HandlePowerSchedule(rrPost, reqPost)
	if rrPost.Code != http.StatusOK {
		t.Fatalf("expected status 200 for POST valid interval, got %d", rrPost.Code)
	}

	var updated power.ScheduleStatus
	if err := json.NewDecoder(rrPost.Body).Decode(&updated); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !updated.Config.Enabled || updated.Config.IntervalHours != 6 {
		t.Errorf("unexpected updated config: %+v", updated.Config)
	}

	// 3. POST Invalid Config (Clock format wrong)
	badCfg := power.ScheduleConfig{
		Enabled:   true,
		Mode:      power.ScheduleModeClock,
		ClockTime: "99:99",
	}
	badBody, _ := json.Marshal(badCfg)
	reqBad := httptest.NewRequest(http.MethodPost, "/api/power/schedule", bytes.NewReader(badBody))
	rrBad := httptest.NewRecorder()
	handlers.HandlePowerSchedule(rrBad, reqBad)
	if rrBad.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid clock, got %d", rrBad.Code)
	}

	// 4. Invalid Method (PUT)
	reqPut := httptest.NewRequest(http.MethodPut, "/api/power/schedule", nil)
	rrPut := httptest.NewRecorder()
	handlers.HandlePowerSchedule(rrPut, reqPut)
	if rrPut.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for PUT, got %d", rrPut.Code)
	}
}
