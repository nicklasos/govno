package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestConvert(t *testing.T) {
	date := time.Now()

	dumpname, _ := Convert("foo/{year}/{month}/{day}/{hours}/{minutes}/{seconds}/{timestamp}")

	if dumpname != "foo/"+fmt.Sprintf("%d/%d/%d/%d/%d/%d/%d", date.Year(), date.Month(), date.Day(), date.Hour(), date.Minute(), date.Second(), date.Unix()) {
		t.Errorf("Dumpname is incorrect: %s", dumpname)
	}

	weekday, _ := Convert("{weekday}")
	_, isoWeekday := date.ISOWeek()
	if weekday != fmt.Sprintf("%d", isoWeekday) {
		t.Error("Weekday in Convert is incorrect")
	}

	week, _ := Convert("{week}")
	if week != fmt.Sprintf("%d", numberOfTheWeek(time.Now())) {
		t.Errorf("Number of the week is incorrent: %s", week)
	}

	host, _ := Convert("{hostname}")
	hostname, _ := os.Hostname()
	if host != hostname {
		t.Errorf("Hostname is incorrect: %s", host)
	}
}

func TestConvertErr(t *testing.T) {
	_, err := Convert("{typo}")
	if err.Error() != "undefined placeholders" {
		t.Error("Detecting typos is not working")
	}
}
