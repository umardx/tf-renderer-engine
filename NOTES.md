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

---

## Part 2 — Terraform Review and Refactor

### Initial Review

Identified concerns:
- Hardcoded values (e.g., region or resource name)
- Lack of provider version
- Flat file structure without modularization
- No explicit environment separation (in case it not segregated by repository or folder)
- Missing or inconsistent tagging strategy
- No backend configuration for remote state

### Before restructuring anything
Run Terraform format and validation
```bash
› tf fmt -recursive
tf validate
╷
│ Error: Unsupported argument
│ 
│   on main.tf line 23, in module "eks":
│   23:   node_groups = {
│ 
│ An argument named "node_groups" is not expected here.
```
The provided terraform are using terraform-aws-modules/eks/aws v19.0.0,
but the configuration is written for v17.
That is a major version jump. In Terraform module: major version = [breaking changes](https://github.com/terraform-aws-modules/terraform-aws-eks/compare/v18.31.2...v19.0.0).

v18+ introduced breaking changes:
instance_type → renamed to instance_types (list)
- Removed `node_groups`
- Introduced `eks_managed_node_groups`
- Introduced `self_managed_node_groups`
- Renamed scaling parameters:
  - `desired_capacity` --> `desired_size`
  - `instance_type` --> `instance_types` (list)

for EKS Managed Node Groups, so I rewrite like this:
```terraform
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "19.0.0"

  cluster_name    = var.cluster_name
  cluster_version = "1.25"

  vpc_id     = var.vpc_id
  subnet_ids = var.subnet_ids

  eks_managed_node_groups = {
    default = {
      desired_size   = 2
      max_size       = 3
      min_size       = 1
      instance_types = ["t3.medium"]
    }
  }

  tags = {
    Environment = var.environment
  }
}
```

Re-validation again, now got this:
```bash
› tf fmt -recursive
tf validate
╷
│ Warning: Argument is deprecated
│ 
│   with aws_s3_bucket.static_assets,
│   on main.tf line 41, in resource "aws_s3_bucket" "static_assets":
│   41:   acl    = "public-read"
│ 
│ Use the aws_s3_bucket_acl resource instead
╵
╷
│ Warning: Deprecated Resource
│ 
│   with module.eks.kubernetes_config_map.aws_auth,
│   on .terraform/modules/eks/main.tf line 498, in resource "kubernetes_config_map" "aws_auth":
│  498: resource "kubernetes_config_map" "aws_auth" {
│ 
│ Deprecated; use kubernetes_config_map_v1.
╵
Success! The configuration is valid, but there were some validation warnings as shown above.
```
In the modern way, the AWS provider split ACL handling into a separate resource. It seems I was running version 4.67.0.

This happened in:
> AWS Provider v4.0.0
> Released: February 2022

Remove acl from the bucket:
```terraform
resource "aws_s3_bucket" "static_assets" {
  bucket = "cluster-static-assets"
  tags = {
    Env = var.environment
  }
}
```
Add this instead:
```terraform
resource "aws_s3_bucket_acl" "static_assets_acl" {
  bucket = aws_s3_bucket.static_assets.id
  acl    = "public-read"
}
```

And quiet important:
```terraform
resource "aws_s3_bucket_public_access_block" "static_assets" {
  bucket = aws_s3_bucket.static_assets.id

  block_public_acls       = false
  block_public_policy     = false
  ignore_public_acls      = false
  restrict_public_buckets = false
}
```
AFAIK if `ignore_public_acls` or `restrict_public_buckets` are true, AWS will override iit.
We must disable all if we want full public exposure.

*Another alternative*, we can use `bucket policy` that define `Sid = "PublicRead"`.
This works even with ownership enforcement.

Next re-validate I got this:
```bash
› tf fmt -recursive
tf validate
╷
│ Warning: Deprecated Resource
│ 
│   with module.eks.kubernetes_config_map.aws_auth,
│   on .terraform/modules/eks/main.tf line 498, in resource "kubernetes_config_map" "aws_auth":
│  498: resource "kubernetes_config_map" "aws_auth" {
│ 
│ Deprecated; use kubernetes_config_map_v1.
╵
Success! The configuration is valid, but there were some validation warnings as shown above.
```
So, Kubernetes provider deprecated `kubernetes_config_map` in favor of `kubernetes_config_map_v1`.
This will eventually break in future provider versions. Right now it works.
We can replace it anyway 😊.

in final, I got no issue:
```bash
› tf -recursive
tf validate
Success! The configuration is valid.
```

### Refactor Strategy

The refactor focused on improving safety, and maintainability.

#### 1. Provider Version
Declare `required_providers` and `required_version` constraints to prevent unintended provider drift and ensure reproduceable.

#### 2. Parameterization
Introduced variables for:
- `aws_region`
- `environment`

This removes hardcoded values and enables environment specific deployments.

> Ignore this if the Terraform configuration is intentionally segregated by repository or folder.

#### 3. Tagging Strategy
Introduced a common tags:
- `environment`
- `managed_by`
- `project`

All resources inherit these tags to improve operational visibility and governance.

#### 4. Backend Considerations
Although not fully implemented for this, in a production setting I would configure:
- S3 backend for remote state
- GCS, S3 or DynamoDB for state locking in AWS

This prevents state corruption and can enable encryption to secure the state in case it stored sensitive state.
