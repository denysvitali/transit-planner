package router

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CalendarEntry represents a service's base weekly schedule from calendar.txt.
// Days is indexed Monday=0..Sunday=6.
type CalendarEntry struct {
	ServiceID string
	Days      [7]bool
	Start     time.Time
	End       time.Time
}

// CalendarException represents a single row from calendar_dates.txt.
// Added=true means exception_type=1 (service added on that date);
// Added=false means exception_type=2 (service removed on that date).
type CalendarException struct {
	ServiceID string
	Date      time.Time
	Added     bool
}

// Calendar holds GTFS service calendar data resolved from calendar.txt
// and calendar_dates.txt.
type Calendar struct {
	Entries    []CalendarEntry
	Exceptions []CalendarException
}

// LoadCalendar reads calendar.txt and calendar_dates.txt from dir. Both files
// are individually optional, but at least one of them must exist.
func LoadCalendar(dir string) (*Calendar, error) {
	calendar := &Calendar{}

	entries, calendarErr := loadCalendarEntries(filepath.Join(dir, "calendar.txt"))
	if calendarErr != nil && !errors.Is(calendarErr, os.ErrNotExist) {
		return nil, calendarErr
	}
	calendar.Entries = entries

	exceptions, exceptionsErr := loadCalendarExceptions(filepath.Join(dir, "calendar_dates.txt"))
	if exceptionsErr != nil && !errors.Is(exceptionsErr, os.ErrNotExist) {
		return nil, exceptionsErr
	}
	calendar.Exceptions = exceptions

	if errors.Is(calendarErr, os.ErrNotExist) && errors.Is(exceptionsErr, os.ErrNotExist) {
		return nil, fmt.Errorf("calendar: neither calendar.txt nor calendar_dates.txt found in %q", dir)
	}

	return calendar, nil
}

// ActiveServicesOn returns the set of service IDs that are running on the
// given calendar date, taking base entries and exceptions into account.
// Only the year/month/day of date are considered.
func (c *Calendar) ActiveServicesOn(date time.Time) map[string]bool {
	active := map[string]bool{}
	day := truncateToDay(date)
	weekday := mondayIndex(day.Weekday())

	for _, entry := range c.Entries {
		if day.Before(truncateToDay(entry.Start)) || day.After(truncateToDay(entry.End)) {
			continue
		}
		if entry.Days[weekday] {
			active[entry.ServiceID] = true
		}
	}

	for _, exception := range c.Exceptions {
		if !truncateToDay(exception.Date).Equal(day) {
			continue
		}
		if exception.Added {
			active[exception.ServiceID] = true
		} else {
			delete(active, exception.ServiceID)
		}
	}

	return active
}

func loadCalendarEntries(path string) ([]CalendarEntry, error) {
	rows, err := readCalendarCSV(path)
	if err != nil {
		return nil, err
	}
	entries := make([]CalendarEntry, 0, len(rows))
	for _, row := range rows {
		start, err := parseGTFSDate(row["start_date"])
		if err != nil {
			return nil, fmt.Errorf("calendar service %q start_date: %w", row["service_id"], err)
		}
		end, err := parseGTFSDate(row["end_date"])
		if err != nil {
			return nil, fmt.Errorf("calendar service %q end_date: %w", row["service_id"], err)
		}
		entry := CalendarEntry{
			ServiceID: row["service_id"],
			Start:     start,
			End:       end,
		}
		dayFields := [7]string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
		for i, name := range dayFields {
			entry.Days[i] = row[name] == "1"
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func loadCalendarExceptions(path string) ([]CalendarException, error) {
	rows, err := readCalendarCSV(path)
	if err != nil {
		return nil, err
	}
	exceptions := make([]CalendarException, 0, len(rows))
	for _, row := range rows {
		date, err := parseGTFSDate(row["date"])
		if err != nil {
			return nil, fmt.Errorf("calendar_dates service %q date: %w", row["service_id"], err)
		}
		exceptions = append(exceptions, CalendarException{
			ServiceID: row["service_id"],
			Date:      date,
			Added:     row["exception_type"] == "1",
		})
	}
	return exceptions, nil
}

// readCalendarCSV is a minimal CSV reader local to the calendar loader so
// the calendar implementation does not depend on internals of gtfs.go.
func readCalendarCSV(path string) ([]map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	for i := range header {
		header[i] = strings.TrimSpace(header[i])
	}

	var rows []map[string]string
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		row := make(map[string]string, len(header))
		for i, name := range header {
			if i < len(record) {
				row[name] = strings.TrimSpace(record[i])
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func parseGTFSDate(value string) (time.Time, error) {
	if len(value) != 8 {
		return time.Time{}, fmt.Errorf("invalid GTFS date %q", value)
	}
	parsed, err := time.ParseInLocation("20060102", value, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid GTFS date %q: %w", value, err)
	}
	return parsed, nil
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// mondayIndex maps time.Weekday (Sunday=0..Saturday=6) to GTFS-style index
// where Monday=0..Sunday=6.
func mondayIndex(w time.Weekday) int {
	if w == time.Sunday {
		return 6
	}
	return int(w) - 1
}
