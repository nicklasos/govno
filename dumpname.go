package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func Convert(name string) string {
	date := time.Now()

	year := fmt.Sprintf("%d", date.Year())
	month := fmt.Sprintf("%d", date.Month())
	day := fmt.Sprintf("%d", date.Day())
	hour := fmt.Sprintf("%d", date.Hour())
	minute := fmt.Sprintf("%d", date.Minute())
	second := fmt.Sprintf("%d", date.Second())
	_, isoWeekday := date.ISOWeek()
	weekday := fmt.Sprintf("%d", isoWeekday)
	week := fmt.Sprintf("%d", numberOfTheWeek(date))
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	replacer := strings.NewReplacer(
		"{year}", year,
		"{month}", month,
		"{day}", day,
		"{hours}", hour,
		"{minutes}", minute,
		"{seconds}", second,
		"{weekday}", weekday,
		"{week}", week,
		"{timestamp}", timestamp,
		"{hostname}", hostname,
	)

	return replacer.Replace(name)
}

func numberOfTheWeek(now time.Time) int {
	beginningOfTheMonth := time.Date(now.Year(), now.Month(), 1, 1, 1, 1, 1, time.UTC)
	_, thisWeek := now.ISOWeek()
	_, beginningWeek := beginningOfTheMonth.ISOWeek()

	return 1 + thisWeek - beginningWeek
}
