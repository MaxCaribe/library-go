// Package api holds the hand-written OpenAPI description of the HTTP API.
// It lives in its own package because go:embed cannot reference files outside
// the directory of the package that declares it.
package api

import _ "embed"

//go:generate go tool oapi-codegen -config cfg.yaml openapi.yaml

//go:embed openapi.yaml
var Spec []byte
