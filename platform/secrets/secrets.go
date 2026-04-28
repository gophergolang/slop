// Package secrets resolves secret references from declared backends.
//
// Reference forms understood by every Store implementation:
//
//	env:OPENAI_API_KEY                — environment variable
//	k8s:namespace/secret-name#key     — Kubernetes Secret
//	vault:path/to/secret#key          — HashiCorp Vault (stubbed; future)
//	awssm:secret-id#json-key          — AWS Secrets Manager (stubbed; future)
//
// The declaration only ever names the reference; resolution happens in code,
// so secret values never appear in vibeguard.yaml.
package secrets

import "context"

// Store resolves a reference to a secret value.
type Store interface {
	Get(ctx context.Context, ref string) (string, error)
}
