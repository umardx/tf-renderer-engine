package api

type RenderRequest struct {
	Payload struct {
		Properties struct {
			AWSRegion  string `json:"aws-region" validate:"required"`
			ACL        string `json:"acl" validate:"required,oneof=private public-read authenticated-read"`
			BucketName string `json:"bucket-name" validate:"required"`
		} `json:"properties" validate:"required"`
	} `json:"payload" validate:"required"`
}
