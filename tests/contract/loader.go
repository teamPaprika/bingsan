// Package contract provides contract testing utilities for validating
// the Bingsan API against the Apache Iceberg REST Catalog OpenAPI spec.
package contract

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// SpecLoader provides utilities for loading and validating against OpenAPI specs.
type SpecLoader struct {
	Doc    *openapi3.T
	Router routers.Router
}

// LoadSpec loads the Iceberg REST Catalog OpenAPI specification.
func LoadSpec() (*SpecLoader, error) {
	specPath := findSpecPath()
	return LoadSpecFromFile(specPath)
}

// LoadSpecFromFile loads an OpenAPI specification from a file path.
func LoadSpecFromFile(path string) (*SpecLoader, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading OpenAPI spec: %w", err)
	}

	// Skip validation - the Iceberg OpenAPI spec has some non-standard elements
	// (like the etag parameter) that kin-openapi doesn't handle well.
	// The spec is validated upstream by the Apache Iceberg project.

	// Override servers with a simple URL to make route matching work.
	// The Iceberg spec uses templated server URLs which don't work well with gorillamux.
	doc.Servers = openapi3.Servers{
		&openapi3.Server{URL: "http://localhost"},
	}

	// Create router for request matching
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("creating router: %w", err)
	}

	return &SpecLoader{
		Doc:    doc,
		Router: router,
	}, nil
}

// ValidateRequest validates an HTTP request against the OpenAPI spec.
func (s *SpecLoader) ValidateRequest(ctx context.Context, input *openapi3filter.RequestValidationInput) error {
	return openapi3filter.ValidateRequest(ctx, input)
}

// ValidateResponse validates an HTTP response against the OpenAPI spec.
func (s *SpecLoader) ValidateResponse(ctx context.Context, input *openapi3filter.ResponseValidationInput) error {
	return openapi3filter.ValidateResponse(ctx, input)
}

// GetOperationByID returns the operation for a given operation ID.
func (s *SpecLoader) GetOperationByID(operationID string) *openapi3.Operation {
	for _, pathItem := range s.Doc.Paths.Map() {
		for _, op := range pathItem.Operations() {
			if op.OperationID == operationID {
				return op
			}
		}
	}
	return nil
}

// ListOperations returns all operation IDs in the spec.
func (s *SpecLoader) ListOperations() []string {
	var ops []string
	for _, pathItem := range s.Doc.Paths.Map() {
		for _, op := range pathItem.Operations() {
			if op.OperationID != "" {
				ops = append(ops, op.OperationID)
			}
		}
	}
	return ops
}

// findSpecPath locates the OpenAPI spec file relative to this source file.
func findSpecPath() string {
	// Try environment variable first
	if path := os.Getenv("ICEBERG_OPENAPI_SPEC"); path != "" {
		return path
	}

	// Get the directory of this source file
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		dir := filepath.Dir(filename)
		specPath := filepath.Join(dir, "iceberg-rest-catalog-open-api.yaml")
		if _, err := os.Stat(specPath); err == nil {
			return specPath
		}
	}

	// Fallback to current directory
	return "iceberg-rest-catalog-open-api.yaml"
}
