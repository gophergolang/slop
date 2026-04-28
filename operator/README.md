# vibeguard-operator

Kubernetes operator that reconciles `Application.vibeguard.dev/v1` resources into:

- NATS JetStream streams + consumers
- Database migrations (`golang-migrate`)
- Workload Deployment + Service + NetworkPolicy + HPA + PDB
- ExternalSecret refs (resolved against a `ClusterSecretStore`)
- Continuous RLS drift detection via `pg_policies` introspection

## Status on branch `4-7`

This module is **scaffolded only**. The CRD types in `api/v1/application_types.go` are real and stable; the reconciler logic in `internal/controller/` is a placeholder. The full implementation is the focus of the follow-up branch — see `../docs/ROADMAP.md`.

The reason for scaffolding now: the CRD types define a contract that the generator (`internal/render/k8s/`) and the LLM layer (`cmd/vibeguard-mcp`'s `query_runtime_state` tool) both depend on. By writing the types now, we lock the contract and can build against it from the rest of the tree.

## Layout (planned)

```
operator/
  PROJECT                                 # kubebuilder layout marker
  api/v1/
    application_types.go                  # CRD types (in this branch)
    groupversion_info.go                  # SchemeBuilder (in this branch)
  cmd/manager/main.go                     # controller-runtime manager (next branch)
  internal/controller/
    application_controller.go             # reconciler (next branch)
  internal/reconcilers/
    migrations.go nats.go workload.go secrets.go rls.go
  internal/webhook/
    application_webhook.go                # validating admission
  config/{crd,manager,rbac}/              # kustomize bundles
  charts/vibeguard-operator/              # Helm chart (supported install path)
  test/{envtest,e2e}/                     # ginkgo + kuttl
```

## Local development (when complete)

```bash
make install            # install CRDs into current kubeconfig context
make run                # run reconciler locally against current context
make docker-build IMG=ghcr.io/vibeguard/operator:dev
make deploy IMG=ghcr.io/vibeguard/operator:dev
```
