# Usage

Start service:
```bash
cd terraform_parse_service
go run main.go
```

Then
```bash
./scripts/render.sh
# or
./scripts/render.sh input.sample.json
```

We expect get:
```bash
Terraform file generated: main.tf
```

If the API returns 400 or 500:
```bash
Request failed with status 400
Response:
{"error":"..."}
```
