package api

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func newHumaAPI(mux *http.ServeMux) huma.API {
	config := huma.DefaultConfig(
		"AdOrbit Authentication API",
		"1.0.0",
	)

	config.Servers = []*huma.Server{
		{
			URL: "/api/v1",
		},
	}

	return humago.New(mux, config)
}
