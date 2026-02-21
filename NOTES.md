# NOTES

## Part 1 — API Service

### Language and Framework Choice

The terraform_parse_service is implemented in Go.

Go was selected because:
- It is suited for building lightweight backend services.
- It produces a single static binary, simplifying containerization.
- The standard library provides sufficient HTTP functionality.
- It integrates cleanly with container or kubernetes environments.

Libraries used:
- `net/http` — HTTP server
- `chi` — routing, just choosen since I want to learn after `gin`
- `go-playground/validator` — structured request validation
- `zap` — structured logging

## Architecture Flow

```
 HTTP JSON
     ↓
    API
     ↓
    spec
(domain model)
     ↓
  renderer
(HCL template)
     ↓
 TF response 
```

The implementation includes:
- Structured input validation
- Controlled error responses
- Panic recovery middleware
- Health endpoint for kube probe

---

### Assumptions
The requirement states: "render to Terraform file (.tf)". So it may require scripts/render.sh on the client side.

Assumption made:
The API returns the generated Terraform configuration as a downloadable `.tf`

This aligns closely with the requirement wording while keeping the service minimal.
