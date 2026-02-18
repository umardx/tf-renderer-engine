package api

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/umardx/tf-renderer-engine/validation"
)

func RenderHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req RenderRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("invalid json", zap.Error(err))
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if err := validation.Validate(req); err != nil {
			logger.Warn("validation failed", zap.Error(err))
			http.Error(w, "validation error: "+err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("validated successfully"))
	}
}
