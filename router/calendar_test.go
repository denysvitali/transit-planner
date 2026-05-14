package router

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestLoadCalendarAndActiveServicesOn(t *testing.T) {
	dir := t.TempDir()

	// weekday_service runs Mon-Fri across all of 2025.
	// weekend_service runs Sat-Sun across all of 2025.
	calendar := "service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date\n" +
		"weekday_service,1,1,1,1,1,0,0,20250101,20251231\n" +
		"weekend_service,0,0,0,0,0,1,1,20250101,20251231\n"
	writeFile(t, filepath.Join(dir, "calendar.txt"), calendar)

	// Exceptions:
	//  - 2025-07-04 (Friday): weekday_service removed (holiday).
	//  - 2025-07-04 (Friday): weekend_service added (special schedule).
	exceptions := "service_id,date,exception_type\n" +
		"weekday_service,20250704,2\n" +
		"weekend_service,20250704,1\n"
	writeFile(t, filepath.Join(dir, "calendar_dates.txt"), exceptions)

	cal, err := LoadCalendar(dir)
	if err != nil {
		t.Fatalf("LoadCalendar: %v", err)
	}
	if len(cal.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(cal.Entries))
	}
	if len(cal.Exceptions) != 2 {
		t.Fatalf("expected 2 exceptions, got %d", len(cal.Exceptions))
	}

	// (a) Regular Wednesday, no exceptions: only weekday_service.
	wednesday := time.Date(2025, 7, 2, 12, 0, 0, 0, time.UTC)
	got := cal.ActiveServicesOn(wednesday)
	if !got["weekday_service"] || got["weekend_service"] {
		t.Fatalf("Wednesday active services = %v; want only weekday_service", got)
	}

	// (b) 2025-07-04 (Friday) has an "added" exception for weekend_service
	//     which normally would not run on Friday.
	holiday := time.Date(2025, 7, 4, 8, 30, 0, 0, time.UTC)
	got = cal.ActiveServicesOn(holiday)
	if !got["weekend_service"] {
		t.Fatalf("holiday should activate weekend_service via added exception, got %v", got)
	}

	// (c) 2025-07-04 (Friday) has a "removed" exception for weekday_service
	//     which would normally run on Friday.
	if got["weekday_service"] {
		t.Fatalf("holiday should deactivate weekday_service via removed exception, got %v", got)
	}

	// Regular Saturday: only weekend_service.
	saturday := time.Date(2025, 7, 5, 0, 0, 0, 0, time.UTC)
	got = cal.ActiveServicesOn(saturday)
	if got["weekday_service"] || !got["weekend_service"] {
		t.Fatalf("Saturday active services = %v; want only weekend_service", got)
	}
}

func TestLoadCalendarOnlyDates(t *testing.T) {
	dir := t.TempDir()

	// Only calendar_dates.txt present (calendar.txt missing).
	exceptions := "service_id,date,exception_type\n" +
		"special,20250704,1\n" +
		"special,20250705,1\n" +
		"special,20250706,2\n"
	writeFile(t, filepath.Join(dir, "calendar_dates.txt"), exceptions)

	cal, err := LoadCalendar(dir)
	if err != nil {
		t.Fatalf("LoadCalendar: %v", err)
	}
	if len(cal.Entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(cal.Entries))
	}
	if len(cal.Exceptions) != 3 {
		t.Fatalf("expected 3 exceptions, got %d", len(cal.Exceptions))
	}

	got := cal.ActiveServicesOn(time.Date(2025, 7, 4, 0, 0, 0, 0, time.UTC))
	if !got["special"] {
		t.Fatalf("expected special active via added exception, got %v", got)
	}

	got = cal.ActiveServicesOn(time.Date(2025, 7, 6, 0, 0, 0, 0, time.UTC))
	if got["special"] {
		t.Fatalf("expected special inactive via removed exception, got %v", got)
	}
}

func TestLoadCalendarMissing(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadCalendar(dir); err == nil {
		t.Fatalf("expected error when neither calendar.txt nor calendar_dates.txt is present")
	}
}
