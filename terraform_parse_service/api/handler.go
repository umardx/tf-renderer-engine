package api

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/umardx/tf-renderer-engine/validation"
	"github.com/umardx/tf-renderer-engine/spec"
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

		bucketSpec := spec.BucketSpec{
			Region:     req.Payload.Properties.AWSRegion,
			ACL:        req.Payload.Properties.ACL,
			BucketName: req.Payload.Properties.BucketName,
		}

		svc := spec.NewService()

		output, err := svc.Render(bucketSpec)
		if err != nil {
			logger.Error("render failed", zap.Error(err))
			http.Error(w, "failed to render terraform template", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(output))
	}
}
