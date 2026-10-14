// Package pathtpl renders backup object keys from templates such as
// "{db}/daily/{weekday}.sql.gz".
package pathtpl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Vars are the non-time values available to a template.
type Vars struct {
	DB       string
	Vno      string
	Hostname string
}

var placeholderRe = regexp.MustCompile(`\{([^{}]*)\}`)

// Placeholders lists every supported placeholder name.
var Placeholders = []string{
	"year", "month", "day", "hours", "minutes", "seconds",
	"weekday", "week", "date", "timestamp", "hostname", "db", "vno",
}

func values(now time.Time, v Vars) map[string]string {
	return map[string]string{
		"year":      fmt.Sprintf("%04d", now.Year()),
		"month":     fmt.Sprintf("%02d", int(now.Month())),
		"day":       fmt.Sprintf("%02d", now.Day()),
		"hours":     fmt.Sprintf("%02d", now.Hour()),
		"minutes":   fmt.Sprintf("%02d", now.Minute()),
		"seconds":   fmt.Sprintf("%02d", now.Second()),
		"weekday":   strconv.Itoa(isoWeekday(now)),
		"week":      strconv.Itoa(WeekOfMonth(now)),
		"date":      now.Format(time.DateOnly),
		"timestamp": strconv.FormatInt(now.Unix(), 10),
		"hostname":  v.Hostname,
		"db":        v.DB,
		"vno":       v.Vno,
	}
}

// Render substitutes all placeholders in tpl. Unknown placeholders and stray
// braces are errors.
func Render(tpl string, now time.Time, v Vars) (string, error) {
	if err := Validate(tpl); err != nil {
		return "", err
	}
	vals := values(now, v)
	return placeholderRe.ReplaceAllStringFunc(tpl, func(m string) string {
		return vals[m[1:len(m)-1]]
	}), nil
}

// Validate checks that tpl contains only known placeholders.
func Validate(tpl string) error {
	if strings.TrimSpace(tpl) == "" {
		return fmt.Errorf("empty path template")
	}
	known := make(map[string]bool, len(Placeholders))
	for _, p := range Placeholders {
		known[p] = true
	}
	for _, m := range placeholderRe.FindAllStringSubmatch(tpl, -1) {
		if !known[m[1]] {
			return fmt.Errorf("path template %q: unknown placeholder {%s}", tpl, m[1])
		}
	}
	if rest := placeholderRe.ReplaceAllString(tpl, ""); strings.ContainsAny(rest, "{}") {
		return fmt.Errorf("path template %q: unbalanced braces", tpl)
	}
	return nil
}

// StaticPrefix returns the part of tpl before the first placeholder.
func StaticPrefix(tpl string) string {
	if i := strings.IndexByte(tpl, '{'); i >= 0 {
		return tpl[:i]
	}
	return tpl
}

// Pattern returns a regexp matching every key tpl can render to for the given
// vars. Time placeholders match any value; {db}, {vno} and {hostname} match
// their literal values.
func Pattern(tpl string, v Vars) (*regexp.Regexp, error) {
	if err := Validate(tpl); err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("^")
	last := 0
	for _, loc := range placeholderRe.FindAllStringSubmatchIndex(tpl, -1) {
		b.WriteString(regexp.QuoteMeta(tpl[last:loc[0]]))
		switch name := tpl[loc[2]:loc[3]]; name {
		case "db":
			b.WriteString(regexp.QuoteMeta(v.DB))
		case "vno":
			b.WriteString(regexp.QuoteMeta(v.Vno))
		case "hostname":
			b.WriteString(regexp.QuoteMeta(v.Hostname))
		case "date":
			b.WriteString(`\d{4}-\d{2}-\d{2}`)
		default:
			b.WriteString(`\d+`)
		}
		last = loc[1]
	}
	b.WriteString(regexp.QuoteMeta(tpl[last:]))
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// isoWeekday returns 1 (Monday) through 7 (Sunday).
func isoWeekday(t time.Time) int {
	if wd := int(t.Weekday()); wd != 0 {
		return wd
	}
	return 7
}

// WeekOfMonth returns the week number within the month (1-6), with weeks
// starting on Monday.
func WeekOfMonth(t time.Time) int {
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	offset := isoWeekday(first) - 1
	return (t.Day()+offset-1)/7 + 1
}
