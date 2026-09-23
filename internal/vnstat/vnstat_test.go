package vnstat_test

import (
	"fmt"
	"testing"
	"time"

	"bfr-webui-go/internal/vnstat"
)

func TestGetTracker(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	tracker := vnstat.GetTracker()
	if tracker == nil {
		t.Fatalf("expected non-nil tracker instance")
	}
}

func TestGetStats(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	stats := vnstat.GetStats()
	if stats.Daily.Interfaces == nil {
		t.Errorf("expected Daily.Interfaces map to be initialized")
	}
	if stats.Yesterday.Interfaces == nil {
		t.Errorf("expected Yesterday.Interfaces map to be initialized")
	}
	if stats.Weekly.Interfaces == nil {
		t.Errorf("expected Weekly.Interfaces map to be initialized")
	}
	if stats.Monthly.Interfaces == nil {
		t.Errorf("expected Monthly.Interfaces map to be initialized")
	}
	if stats.DailyHistory == nil {
		t.Errorf("expected DailyHistory slice to be initialized")
	}
}

func TestResetStats(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	tracker := vnstat.GetTracker()
	tracker.AddTrafficDelta("wlan0", 5000, 2000)

	stats := tracker.ResetStats()
	if stats.Daily.Interfaces == nil {
		t.Errorf("expected Daily.Interfaces map to be initialized after reset")
	}
	if stats.Yesterday.Interfaces == nil {
		t.Errorf("expected Yesterday.Interfaces map to be initialized after reset")
	}
	if stats.Weekly.Interfaces == nil {
		t.Errorf("expected Weekly.Interfaces map to be initialized after reset")
	}
	if stats.Monthly.Interfaces == nil {
		t.Errorf("expected Monthly.Interfaces map to be initialized after reset")
	}
	if len(stats.DailyHistory) != 0 {
		t.Errorf("expected DailyHistory to be empty after reset, got %d items", len(stats.DailyHistory))
	}
}

func TestRollover_DayWeekMonth(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	tracker := vnstat.GetTracker()

	// Initial time: Sunday, Sep 20, 2026 (2026-W38)
	t0, _ := time.Parse("2006-01-02 15:04:05", "2026-09-20 12:00:00")
	tracker.ResetStatsAt(t0)

	// Add traffic on Sep 20
	tracker.AddTrafficDelta("wlan0", 1000, 500)
	stats0 := tracker.GetSnapshot()
	if stats0.Daily.Interfaces["wlan0"].RxBytes < 1000 {
		t.Errorf("expected at least 1000 RxBytes, got %d", stats0.Daily.Interfaces["wlan0"].RxBytes)
	}

	// Rollover to Monday, Sep 21, 2026 (new day and new ISO week 2026-W39)
	t1, _ := time.Parse("2006-01-02 15:04:05", "2026-09-21 08:00:00")
	tracker.CheckRollover(t1)

	stats1 := tracker.GetSnapshot()

	// Yesterday should have the 1500 total bytes from Sep 20
	if stats1.Yesterday.Total.Total != 1500 {
		t.Errorf("expected Yesterday total to be 1500, got %d", stats1.Yesterday.Total.Total)
	}

	// DailyHistory should have 1 record with date 2026-09-20
	if len(stats1.DailyHistory) != 1 {
		t.Fatalf("expected 1 DailyHistory entry, got %d", len(stats1.DailyHistory))
	}
	if stats1.DailyHistory[0].Date != "2026-09-20" {
		t.Errorf("expected DailyHistory date 2026-09-20, got %s", stats1.DailyHistory[0].Date)
	}
	if stats1.DailyHistory[0].Total != 1500 {
		t.Errorf("expected DailyHistory total 1500, got %d", stats1.DailyHistory[0].Total)
	}

	// Weekly should have reset due to ISO week change
	if stats1.Weekly.Total.Total != 0 {
		t.Errorf("expected Weekly total to reset to 0, got %d", stats1.Weekly.Total.Total)
	}

	// Monthly should still retain the 1500 bytes from September
	if stats1.Monthly.Total.Total < 1500 {
		t.Errorf("expected Monthly total to be >= 1500, got %d", stats1.Monthly.Total.Total)
	}

	// Rollover to next month: Oct 01, 2026
	t2, _ := time.Parse("2006-01-02 15:04:05", "2026-10-01 08:00:00")
	tracker.CheckRollover(t2)

	stats2 := tracker.GetSnapshot()
	// Monthly should have reset on month change
	if stats2.Monthly.Total.Total != 0 {
		t.Errorf("expected Monthly total to reset to 0 in new month, got %d", stats2.Monthly.Total.Total)
	}
}

func TestDailyHistory_MaxSevenDays(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	tracker := vnstat.GetTracker()

	day1, _ := time.Parse("2006-01-02 15:04:05", "2026-01-01 12:00:00")
	tracker.ResetStatsAt(day1)
	tracker.AddTrafficDelta("wlan0", 100, 100)

	// Simulate remaining 9 days of rollover (days 2 to 10)
	for i := 2; i <= 10; i++ {
		dateStr := fmt.Sprintf("2026-01-%02d 12:00:00", i)
		dayTime, _ := time.Parse("2006-01-02 15:04:05", dateStr)
		tracker.CheckRollover(dayTime)
		tracker.AddTrafficDelta("wlan0", uint64(i*100), uint64(i*100))
	}

	// Final rollover to 11th day to push 10th day into history
	endDay, _ := time.Parse("2006-01-02 15:04:05", "2026-01-11 12:00:00")
	tracker.CheckRollover(endDay)

	stats := tracker.GetSnapshot()
	if len(stats.DailyHistory) > 7 {
		t.Errorf("expected max 7 DailyHistory entries, got %d", len(stats.DailyHistory))
	}
	if len(stats.DailyHistory) == 7 {
		// Oldest day in history should be 2026-01-04 (days 4 through 10)
		if stats.DailyHistory[0].Date != "2026-01-04" {
			t.Errorf("expected oldest history date to be 2026-01-04, got %s", stats.DailyHistory[0].Date)
		}
		if stats.DailyHistory[6].Date != "2026-01-10" {
			t.Errorf("expected newest history date to be 2026-01-10, got %s", stats.DailyHistory[6].Date)
		}
	}
}
