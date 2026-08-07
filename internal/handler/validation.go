package handler

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/robfig/cron/v3"
)

const maxNameLength = 100

// scheduleParser mirrors the parser used by the polling service
// (cron.New(cron.WithSeconds())): 6-field specs plus descriptors like @every,
// @daily, @hourly.
var scheduleParser = cron.NewParser(
	cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name is required")
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return fmt.Errorf("name must be at most %d characters", maxNameLength)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("name must not contain control characters")
		}
	}
	return nil
}

func validateSchedule(schedule string) error {
	if strings.TrimSpace(schedule) == "" {
		return fmt.Errorf("schedule is required")
	}
	if _, err := scheduleParser.Parse(schedule); err != nil {
		return fmt.Errorf("invalid schedule format: %v", err)
	}
	return nil
}

func validateURL(rawURL string) error {
	if strings.TrimSpace(rawURL) == "" {
		return fmt.Errorf("url is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %v", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url must use the http or https scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("url must include a host")
	}
	return nil
}
