package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func Convert(name string) string {
	date := time.Now()

	_, isoWeekday := date.ISOWeek()
	weekday := fmt.Sprintf("%d", isoWeekday)

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	replacer := strings.NewReplacer(
		"{year}", fmt.Sprintf("%d", date.Year()),
		"{month}", fmt.Sprintf("%d", date.Month()),
		"{day}", fmt.Sprintf("%d", date.Day()),
		"{hours}", fmt.Sprintf("%d", date.Hour()),
		"{minutes}", fmt.Sprintf("%d", date.Minute()),
		"{seconds}", fmt.Sprintf("%d", date.Second()),
		"{weekday}", weekday,
		"{week}", fmt.Sprintf("%d", numberOfTheWeek(date)),
		"{timestamp}", fmt.Sprintf("%d", time.Now().Unix()),
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
