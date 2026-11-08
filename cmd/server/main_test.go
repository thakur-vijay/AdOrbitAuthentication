package main

import (
	"reflect"
	"testing"

	swaggerDocs "github.com/adorbit/authentication/docs"
)

func TestSetupSwaggerUsesDefaultHTTPHostAndConfiguredHTTPSHost(t *testing.T) {
	oldHost := swaggerDocs.SwaggerInfo.Host
	oldBasePath := swaggerDocs.SwaggerInfo.BasePath
	oldSchemes := append([]string(nil), swaggerDocs.SwaggerInfo.Schemes...)
	t.Cleanup(func() {
		swaggerDocs.SwaggerInfo.Host = oldHost
		swaggerDocs.SwaggerInfo.BasePath = oldBasePath
		swaggerDocs.SwaggerInfo.Schemes = oldSchemes
	})

	setupSwagger("", ":8080")
	if swaggerDocs.SwaggerInfo.Host != "localhost:8081" || swaggerDocs.SwaggerInfo.BasePath != "/api/v1" || !reflect.DeepEqual(swaggerDocs.SwaggerInfo.Schemes, []string{"http"}) {
		t.Fatalf("default Swagger endpoint = host %q base %q schemes %#v", swaggerDocs.SwaggerInfo.Host, swaggerDocs.SwaggerInfo.BasePath, swaggerDocs.SwaggerInfo.Schemes)
	}
	setupSwagger("api.example.test", ":8080")
	if swaggerDocs.SwaggerInfo.Host != "api.example.test" || !reflect.DeepEqual(swaggerDocs.SwaggerInfo.Schemes, []string{"https"}) {
		t.Fatalf("configured Swagger endpoint = host %q schemes %#v", swaggerDocs.SwaggerInfo.Host, swaggerDocs.SwaggerInfo.Schemes)
	}
}
