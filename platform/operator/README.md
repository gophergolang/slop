# VibeGuard App Operator (Lightweight)

This is a **lightweight operator** for VibeGuard applications.

## Current Implementation (GitOps-based)

Instead of a full Kubernetes Operator (which requires controller-runtime, CRDs, etc.), we start with a **GitOps-native approach** using:

- Argo CD Application per app
- Kustomize overlays per environment
- The declaration (`vibeguard.yaml`) drives the generation of these manifests

## Generated Structure

```
apps/
  team-task-saas/
    base/
      deployment.yaml
      service.yaml
      nats-consumers.yaml
      networkpolicy.yaml
    overlays/
      dev/
      staging/
      prod/
    kustomization.yaml
```

## Future Evolution

When ready, we can upgrade to a real Kubernetes Operator that:
- Watches `VibeGuardApp` CRDs
- Automatically generates the above manifests
- Manages NATS consumers as custom resources
- Enforces security policies from the declaration at runtime

This keeps the initial implementation simple while being fully compatible with the future operator model.