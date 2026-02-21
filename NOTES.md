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

---

## Part 3 — Helm Chart Review and Fixes

### 1. Initial Deployment Attempt
```bash
helm install my-app ./helm --dry-run
...
templates/backend-deployment.yaml: error converting YAML to JSON: yaml: line 2: mapping values are not allowed in this context
...
```

There was issue on YAML. I fix it by removing unatend character `\`
Anyway, I can see is that the selector field is missing (spotted by my linter LOL), which is required.
Add this line into `backend-deployment.yaml`:
```yaml
...
spec:
  replicas: 1
  selector:
    matchLabels:
      app: backend
...
```

Add this line into `frontend-deployment.yaml` too:
```yaml
...
spec:
  replicas: 2
  selector:
    matchLabels:
      app: frontend-app
...
```

From this, I became interested in checking the service as well, and voila: the selector `backend-service.yaml` is fine.
Fix `frontend-service.yaml`:
```yaml
  selector:
    app: frontend-app
```

Without this fix, this resulted in:
- Service having zero endpoints
- Traffic not routing

Alternative **Fix:**
Standardized labels and selectors to use consistent naming via Helm templates:

labels:
```
app: {{ .Values.backend.name }}
```

selector:
```
app: {{ .Values.backend.name }}
```

### 2. Hardcoded Values

So templates must respect that exact structure based on values.yaml
We aligned exactly to:
```
.Values.replicaCount
.Values.backend.image.repository
.Values.backend.service.port
.Values.resources
```

As result, we can deploy it in dry-run:
```
NAME: my-app
LAST DEPLOYED: Sun Feb 22 02:30:36 2026
NAMESPACE: local
STATUS: pending-install
REVISION: 1
TEST SUITE: None
HOOKS:
MANIFEST:
---
# Source: my-apps/templates/backend-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: backend
spec:
  type: ClusterIP
  selector:
    app: backend
  ports:
    - port: 8080
      targetPort: 8080
      protocol: TCP
---
# Source: my-apps/templates/frontend-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: frontend
spec:
  type: LoadBalancer
  selector:
    app: frontend
  ports:
    - port: 80
      targetPort: 80
      protocol: TCP
---
# Source: my-apps/templates/backend-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: backend
  template:
    metadata:
      labels:
        app: backend
    spec:
      containers:
        - name: backend
          image: "hashicorp/http-echo:0.2.3"
          ports:
            - containerPort: 8080
          resources:
            {}
---
# Source: my-apps/templates/frontend-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
        - name: frontend
          image: "nginx:1.16.0"
          ports:
            - containerPort: 80
          resources:
            {}
---
# Source: my-apps/templates/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: backend-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: backend
  minReplicas: 1
  maxReplicas: 5
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 50
```

Validate using EndpointSlice
```
› kg EndpointSlice
NAME             ADDRESSTYPE   PORTS   ENDPOINTS      AGE
backend-pw587    IPv4          8080    10.222.0.251   42s
frontend-kw6vc   IPv4          80      10.222.1.95    42s
```

---

## Part 4 — System Behavior Under Load and Failure

> I assume this regarding helm chart?

Under moderate load, behavior will depend on:
1. Backend CPU and memory limits
2. Whether HPA is enabled
3. Kubernetes node capacity
4. Network latency between services

when HPA is disabled, scaling is static and depend on `replicaCount`.
This means the backend could become CPU under pressure when increased request volume.

If HPA is enabled and kube metrics server is available, scaling will happen based on CPU utilization. However, scaling speed time depends on:

- Pod resource requests config
- Metrics interval
- Startup time pod

Let say, in Java, startup can be so long within minutes. While in Golang, it fast.


### Failure Scenarios
#### 1. Let say a backend pod crashes:

1. Kubernetes Deployment automatically recreates it.
2. Liveness probes help detect unresponsive containers then restart it.
3. Readiness probes prevent traffic routing to unhealthy pods (EndPointSlice).

While in helm chart templates, liveness and readiness probes are not set, this can be 5xx issues anytime from the client.

And without PDB, voluntary disruptions (e.g., node drain) may reduce availability without notice.


#### 2. Let say Kubernetes node fails

1. Pods are rescheduled to the other nodes (if capacity available).
2. Recovery the cluster autoscalar will remove the bad node and adding new node.

Without cluster autoscaler new pods forever pending if no node available or fit by the pod resource request.

#### 3. Let say we are facing a high traffic surge

When traffic spikes suddenly:
1. Without HPA: client requests may queue or time out and CPU saturated.
2. With HPA: pods scale based on CPU utilization, but scaling is reactive. Initial start latency may temporarily degrade performance. Sometimes we must set max scaling to keep DB connection pool stable.

Common mitigation strategies:
1. Configure HPA with appropriate thresholds. Each service, may different configuration
2. Right-sizing CPU requests
3. Consider pre-warming replicas during predictable traffic spikes

Sometime business allow it, we can do circuit breaker or apply rate limiter at the ingress layer.

### Long Term Resilience

1. Enable CPU based HPA with properly defined resource requests.
2. Ensure minimum availability during voluntary disruptions with PDB.
3. Enable autoscalar node scaling under increased demand.
4. Observability and Alerting
Earlier detection with analyzing log/metric and alert can prevent outage.
5. Avoid `latest` tags in production like values.yaml provided in the test.
Use immutable version tags. I know this is hard and causing headache when troubleshooting.

---

## Part 5 — Approach & Tools
### Overall Approach
I approached the assignment incrementally, before optimization, I prefer to see what the common practice and stability.

1. Ensured each component worked independently:
   - API service compiled and responded correctly.
   - Terraform configuration validated and produced a successful over tf plan.
   - Helm chart valid when dry-run.

2. Identified breaking issues before refactoring:
   - Detected Terraform module version mismatch (EKS v19 changes).
   - Validated deployment (label/selector) match with its service.
   - Verified Service EndpointSlice.

The guiding principle was:  
**Make it work → Make it correct → Make it resilient.**
> For helm chart template, I will make it resilient later LOL.

---

### Terraform Investigation
To resolve the EKS module issue:

- Terraform init and validate
- Reviewed the `terraform-aws-modules/eks` in GitHub repository.
- Examined the CHANGELOG for breaking changes in v18/v19.
- Confirmed removal of `node_groups` and migration to `eks_managed_node_groups`.

This ensured compatibility with the pinned module version instead of downgrading blindly.

---

### Kubernetes & Helm Debugging
To debug Helm deployment:

- Used `helm lint` and `helm template` to inspect rendered manifests. Or we can direcly use `helm install --dry-run`
- Checked Service routing using `kubectl get EndpointSlice`
- Later once deployed, used `kubectl describe service` to confirm selector alignment.
- Used `kubectl port-forward` for functional validation.

The main focus was validating label/selector consistency and correct port mapping.

---

### Go API Development

The API service was implemented in Go using:
- `net/http` for HTTP handling
- `chi` for routing
- `zap` for structured logging
- `go-playground/validator` for request validation

I used:
- Official Go documentation
- Library documentation for `chi` and `zap`
- Cursor AI for exploring usage patterns and understanding logging `zap` best practices. It help me to improove grammer on creating README.md and NOTES.md as well.
