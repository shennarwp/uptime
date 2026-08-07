package handler

import (
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantMsg string
	}{
		{name: "plain", value: "My Target"},
		{name: "special characters", value: "Östlich & co (prod)#1"},
		{name: "leading and trailing spaces", value: "  My Target  "},
		{name: "exactly max length (ascii)", value: strings.Repeat("a", maxNameLength)},
		{name: "exactly max length (unicode)", value: strings.Repeat("😀", maxNameLength)},
		{name: "empty", value: "", wantMsg: "name is required"},
		{name: "whitespace only", value: "   \t ", wantMsg: "name is required"},
		{name: "too long (ascii)", value: strings.Repeat("a", maxNameLength+1), wantMsg: "at most 100 characters"},
		{name: "too long (unicode)", value: strings.Repeat("😀", maxNameLength+1), wantMsg: "at most 100 characters"},
		{name: "newline control character", value: "Bad\nName", wantMsg: "control characters"},
		{name: "tab control character", value: "Bad\tName", wantMsg: "control characters"},
		{name: "null byte", value: "Bad\x00Name", wantMsg: "control characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateName(tt.value)
			if tt.wantMsg == "" {
				if err != nil {
					t.Errorf("unexpected error for %q: %v", tt.value, err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error for %q, got nil", tt.value)
				return
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("expected error %q to contain %q, got %q", tt.value, tt.wantMsg, err.Error())
			}
		})
	}
}

func TestValidateSchedule(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantMsg string
	}{
		{name: "6-field cron", value: "0 0 */3 * * *"},
		{name: "6-field cron with step", value: "*/5 * * * * *"},
		{name: "every second", value: "* * * * * *"},
		{name: "descriptor @every", value: "@every 5m"},
		{name: "descriptor @every hours", value: "@every 1h"},
		{name: "descriptor @hourly", value: "@hourly"},
		{name: "descriptor @daily", value: "@daily"},
		{name: "empty", value: "", wantMsg: "schedule is required"},
		{name: "whitespace only", value: "   ", wantMsg: "schedule is required"},
		{name: "not a cron spec", value: "not-a-cron", wantMsg: "invalid schedule format"},
		{name: "5-field (missing seconds)", value: "0 * * * *", wantMsg: "invalid schedule format"},
		{name: "7-field", value: "* * * * * * *", wantMsg: "invalid schedule format"},
		{name: "descriptor without value", value: "@every", wantMsg: "invalid schedule format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSchedule(tt.value)
			if tt.wantMsg == "" {
				if err != nil {
					t.Errorf("unexpected error for %q: %v", tt.value, err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error for %q, got nil", tt.value)
				return
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("expected error %q to contain %q, got %q", tt.value, tt.wantMsg, err.Error())
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantMsg string
	}{
		{name: "https", value: "https://example.com"},
		{name: "http", value: "http://example.com"},
		{name: "https with path and query", value: "https://example.com/api/v1?x=1&y=2"},
		{name: "https with custom port", value: "https://example.com:8443/path"},
		{name: "http localhost", value: "http://localhost:8080"},
		{name: "https subdomain", value: "https://sub.domain.example.co.uk/x"},
		{name: "https with userinfo", value: "https://user@example.com"},
		{name: "empty", value: "", wantMsg: "url is required"},
		{name: "whitespace only", value: "   ", wantMsg: "url is required"},
		{name: "no scheme", value: "example.com", wantMsg: "http or https scheme"},
		{name: "scheme-relative", value: "//example.com", wantMsg: "http or https scheme"},
		{name: "ftp scheme", value: "ftp://example.com", wantMsg: "http or https scheme"},
		{name: "javascript scheme", value: "javascript:alert(1)", wantMsg: "http or https scheme"},
		{name: "relative path (no scheme)", value: "not a url", wantMsg: "http or https scheme"},
		{name: "unparseable escape", value: "https://example.com/%zz", wantMsg: "invalid url"},
		{name: "space in host", value: "http://exa mple.com", wantMsg: "invalid url"},
		{name: "missing host", value: "https://", wantMsg: "must include a host"},
		{name: "missing host with path", value: "http:///path", wantMsg: "must include a host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.value)
			if tt.wantMsg == "" {
				if err != nil {
					t.Errorf("unexpected error for %q: %v", tt.value, err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error for %q, got nil", tt.value)
				return
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("expected error %q to contain %q, got %q", tt.value, tt.wantMsg, err.Error())
			}
		})
	}
}
