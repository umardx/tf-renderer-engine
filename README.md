# tf-renderer-engine

This service accepts an API request and renders a Terraform `.tf` file that provisions an AWS S3 bucket.

## Overview

The service:
- Receives a POST request with S3 configuration parameters
- Validates the payload
- Renders a Terraform configuration file
- Returns the generated `.tf` file as a downloadable response

The generated Terraform configuration includes:
- `provider "aws"`
- `aws_s3_bucket`
- `aws_s3_bucket_acl`


## Run Locally

```
cd terraform_parse_service
go mod init github.com/umardx/tf-renderer-engine
go mod tidy

go run main.go
```
Default port: `8080`


### Health Check
```bash
curl http://localhost:8080/healthz
```

### Bash Script Helper
A helper script is provided:
```
./scripts/render.sh \
   ./scripts/input.sample.json
```
It automatically saves main.tf on success and prints the error response on failure.
