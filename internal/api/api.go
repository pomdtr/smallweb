package api

import (
	"encoding/json"
	"net/http"
)

// NewApi creates an http.Handler with the API endpoints and OpenAPI spec endpoint
func NewApi(rootDir string) http.Handler {
	mux := http.NewServeMux()

	// Add the main API handler
	apiHandler := NewHandler(rootDir)
	mux.Handle("/v1/", apiHandler)

	// Add the OpenAPI spec endpoint
	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
		swagger, err := GetSwagger()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(swagger)
	})

	return mux
}
