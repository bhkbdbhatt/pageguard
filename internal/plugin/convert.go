// Package plugin hosts the validator/reporter/integration engine. It evaluates
// built-in declarative rules and external plugins against page observations.
package plugin

import (
	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/pkg/pluginsdk"
)

// ToSDK converts an internal observation into the public plugin wire type.
func ToSDK(o *domain.Observation) *pluginsdk.Observation {
	if o == nil {
		return nil
	}
	obs := &pluginsdk.Observation{
		URL:           o.URL,
		FinalURL:      o.FinalURL,
		StatusCode:    o.StatusCode,
		ContentType:   o.ContentType,
		Title:         o.Title,
		Headers:       o.Headers,
		HTML:          o.HTML,
		HTMLTruncated: o.HTMLTruncated,
		Rendered:      o.Rendered,
		Depth:         o.Depth,
		PageID:        o.PageID.String(),
	}
	for _, r := range o.Requests {
		obs.Requests = append(obs.Requests, pluginsdk.NetworkRequest{
			Method: r.Method, URL: r.URL, Host: r.Host, Status: r.Status,
			ResourceType: r.ResourceType, Initiator: r.Initiator,
			RequestHeaders: r.RequestHeaders, ResponseHeaders: r.ResponseHeaders,
			DurationMs: r.DurationMs, SizeBytes: r.SizeBytes, ThirdParty: r.ThirdParty,
		})
	}
	for _, c := range o.Cookies {
		obs.Cookies = append(obs.Cookies, pluginsdk.Cookie{
			Name: c.Name, Domain: c.Domain, Path: c.Path, ValueHash: c.ValueHash,
			Secure: c.Secure, HTTPOnly: c.HTTPOnly, SameSite: c.SameSite,
			ExpiresUnix: c.ExpiresUnix, Session: c.Session, FirstParty: c.FirstParty,
		})
	}
	for _, t := range o.Tags {
		obs.Tags = append(obs.Tags, pluginsdk.Tag{
			ID: t.ID, Vendor: t.Vendor, Name: t.Name, Category: string(t.Category),
			Version: t.Version, Evidence: t.Evidence, Confidence: t.Confidence,
		})
	}
	for _, c := range o.Console {
		if c.Level == "error" {
			obs.ConsoleErrors = append(obs.ConsoleErrors, c.Text)
		}
	}
	obs.Geolocation = o.Geolocation
	for _, d := range o.DataLayer {
		obs.DataLayer = append(obs.DataLayer, pluginsdk.DataLayerEntry{
			Name:  d.Name,
			Value: string(d.Value),
		})
	}
	return obs
}
