package api

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/umardx/tf-renderer-engine/spec"
	"github.com/umardx/tf-renderer-engine/validation"
)

func RenderHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req RenderRequest

		// Decode JSON body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("invalid json body", zap.Error(err))
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: "invalid request body",
			})
			return
		}

		// Validate request
		if err := validation.Validate(req); err != nil {
			logger.Warn("validation failed", zap.Error(err))
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error: err.Error(),
			})
			return
		}

		bucketSpec := spec.BucketSpec{
			Region:     req.Payload.Properties.AWSRegion,
			ACL:        req.Payload.Properties.ACL,
			BucketName: req.Payload.Properties.BucketName,
		}

		// Render TF spec
		svc := spec.NewService()
		output, err := svc.Render(bucketSpec)
		if err != nil {
			logger.Error("render failed", zap.Error(err))
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{
				Error: "failed to render terraform configuration",
			})
			return
		}

		logger.Info("terraform rendered successfully",
			zap.String("bucket", bucketSpec.BucketName),
			zap.String("region", bucketSpec.Region),
		)

		// Return raw TF file
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(output))
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
