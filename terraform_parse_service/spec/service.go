package spec

import "github.com/umardx/tf-renderer-engine/renderer"

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Render(spec BucketSpec) (string, error) {
	return renderer.RenderS3Bucket(spec)
}
