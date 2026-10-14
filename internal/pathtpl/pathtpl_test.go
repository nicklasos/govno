package pathtpl

import (
	"testing"
	"time"
)

var vars = Vars{DB: "shop", Vno: "daily", Hostname: "web-1"}

func TestRender(t *testing.T) {
	now := time.Date(2026, time.March, 5, 7, 8, 9, 0, time.UTC) // Thursday

	tests := []struct {
		tpl  string
		want string
	}{
		{"{year}/{month}/{day}/{hours}{minutes}{seconds}.sql.gz", "2026/03/05/070809.sql.gz"},
		{"{db}/{vno}/{weekday}.sql.gz", "shop/daily/4.sql.gz"},
		{"{hostname}/{date}.sql.gz", "web-1/2026-03-05.sql.gz"},
		{"{timestamp}", "1772694489"},
		{"{week}", "2"}, // March 1 is a Sunday
		{"static.sql.gz", "static.sql.gz"},
	}
	for _, tt := range tests {
		got, err := Render(tt.tpl, now, vars)
		if err != nil {
			t.Fatalf("Render(%q): %v", tt.tpl, err)
		}
		if got != tt.want {
			t.Errorf("Render(%q) = %q, want %q", tt.tpl, got, tt.want)
		}
	}
}

func TestRenderErrors(t *testing.T) {
	for _, tpl := range []string{"", "{typo}", "{host}/x", "a{b", "a}b", "{year"} {
		if _, err := Render(tpl, time.Now(), vars); err == nil {
			t.Errorf("Render(%q): expected error", tpl)
		}
	}
}

func TestWeekOfMonth(t *testing.T) {
	tests := []struct {
		date string
		want int
	}{
		{"2026-01-01", 1}, // Thursday, ISO week 1
		{"2026-01-04", 1}, // Sunday
		{"2026-01-05", 2}, // Monday
		{"2027-01-01", 1}, // Friday, ISO week 53 of 2026
		{"2027-01-04", 2},
		{"2025-12-29", 5}, // Monday, ISO week 1 of 2026
		{"2025-12-31", 5},
		{"2026-03-01", 1}, // month starting on Sunday
		{"2026-03-02", 2},
		{"2026-03-31", 6},
		{"2026-06-01", 1}, // month starting on Monday
		{"2026-06-07", 1},
		{"2026-06-08", 2},
	}
	for _, tt := range tests {
		d, _ := time.Parse(time.DateOnly, tt.date)
		if got := WeekOfMonth(d); got != tt.want {
			t.Errorf("WeekOfMonth(%s) = %d, want %d", tt.date, got, tt.want)
		}
	}
}

func TestWeekdaySunday(t *testing.T) {
	sunday := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	got, _ := Render("{weekday}", sunday, vars)
	if got != "7" {
		t.Errorf("Sunday weekday = %q, want 7", got)
	}
}

func TestStaticPrefix(t *testing.T) {
	tests := map[string]string{
		"mysql/{db}/{day}.gz": "mysql/",
		"{db}/x":              "",
		"no/placeholders.gz":  "no/placeholders.gz",
	}
	for tpl, want := range tests {
		if got := StaticPrefix(tpl); got != want {
			t.Errorf("StaticPrefix(%q) = %q, want %q", tpl, got, want)
		}
	}
}

func TestPattern(t *testing.T) {
	tpl := "backups/{db}/{vno}/{date}-{hours}.sql.gz"
	re, err := Pattern(tpl, vars)
	if err != nil {
		t.Fatal(err)
	}
	rendered, _ := Render(tpl, time.Now(), vars)
	matches := []string{rendered, "backups/shop/daily/2020-01-02-03.sql.gz"}
	misses := []string{
		"backups/other/daily/2020-01-02-03.sql.gz",
		"backups/shop/weekly/2020-01-02-03.sql.gz",
		"backups/shop/daily/2020-01-02-03.sql.gz.bak",
		"backups/shop/daily/2020-01-02-xx.sql.gz",
		"backupsXshop/daily/2020-01-02-03.sql.gz",
	}
	for _, k := range matches {
		if !re.MatchString(k) {
			t.Errorf("Pattern should match %q", k)
		}
	}
	for _, k := range misses {
		if re.MatchString(k) {
			t.Errorf("Pattern should not match %q", k)
		}
	}
}
