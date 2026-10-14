package routeApi

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"

	"github.com/yusing/godoxy/internal/route"
	"github.com/yusing/godoxy/internal/routing"
)

type RouteType struct {
	route.Route
	PublicURLs []string `json:"public_urls,omitempty" extensions:"x-omitempty"` // Public HTTP URLs derived from runtime domains and listeners
} // @name RouteWithPublicURLs

type routeResponse struct {
	Route      routing.Route
	PublicURLs []string
}

func (r routeResponse) MarshalJSONTo(enc *jsontext.Encoder) error {
	// Preserve the consuming encoder's route payload: Gin uses v1 options,
	// while WebSocket updates use v2 options.
	data, err := json.Marshal(r.Route, enc.Options())
	if err != nil {
		return err
	}
	return json.MarshalEncode(enc, &struct {
		Route      jsontext.Value `json:",embed"`
		PublicURLs []string       `json:"public_urls,omitempty"`
	}{data, r.PublicURLs})
}
