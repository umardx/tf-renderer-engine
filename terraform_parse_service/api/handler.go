package api

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

func RenderHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req map[string]interface{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("invalid json", zap.Error(err))
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("render logic not implemented yet"))
	}
}
