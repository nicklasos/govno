package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

func Convert(name string) (string, error) {
	date := time.Now()

	_, isoWeekday := date.ISOWeek()
	weekday := fmt.Sprintf("%d", isoWeekday)

	hostname, err := os.Hostname()
	if err != nil {
		return "", err
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

	result := replacer.Replace(name)

	if strings.ContainsAny(result, "{}") {
		return "", errors.New("Undefined placeholders")
	}

	return result, nil
}

func numberOfTheWeek(now time.Time) int {
	beginningOfTheMonth := time.Date(now.Year(), now.Month(), 1, 1, 1, 1, 1, time.UTC)
	_, thisWeek := now.ISOWeek()
	_, beginningWeek := beginningOfTheMonth.ISOWeek()

	return 1 + thisWeek - beginningWeek
}
