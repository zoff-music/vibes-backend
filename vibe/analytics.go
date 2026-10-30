package vibe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// AnalyticsEvent follows the official Plausible browser tracker's wire format.
type AnalyticsEvent struct {
	Name        string                     `json:"n"`
	URL         string                     `json:"u"`
	Domain      string                     `json:"d"`
	Referrer    string                     `json:"r,omitempty"`
	Properties  map[string]json.RawMessage `json:"p,omitempty"`
	Interactive *bool                      `json:"i,omitempty"`
	Version     int                        `json:"v,omitempty"`
	HashMode    int                        `json:"h,omitempty"`
	ScrollDepth int                        `json:"sd,omitempty"`
	Engagement  int64                      `json:"e,omitempty"`
	Revenue     json.RawMessage            `json:"$,omitempty"`
}

func (e *AnalyticsEvent) Validate(domain string) error {
	if e.Domain != domain || strings.TrimSpace(e.Name) == "" || len(e.Name) > 120 {
		return fmt.Errorf("error validating analytics event: invalid domain or name")
	}
	page, err := url.Parse(e.URL)
	if err != nil {
		return fmt.Errorf("error parsing analytics URL: %w", err)
	}
	if page.Scheme != "https" || page.Host != domain || page.User != nil {
		return fmt.Errorf("error validating analytics URL: invalid origin")
	}
	return nil
}

type AnalyticsVisitor struct {
	IP        string
	UserAgent string
}

type AnalyticsResult struct {
	Dropped bool
}

type AnalyticsEventCreator interface {
	CreateAnalyticsEvent(ctx context.Context, event AnalyticsEvent, visitor AnalyticsVisitor) (*AnalyticsResult, error)
}
