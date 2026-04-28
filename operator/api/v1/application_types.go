// Package v1: Application CRD types.
//
// This file defines the Go types that map to the vibeguard.dev/v1 Application
// CRD. The spec mirrors the vibeguard.yaml declaration so the operator and
// the generator share one schema.
//
// Status: scaffolded in branch 4-7. The reconciler that consumes these types
// is in internal/controller (also scaffolded — full reconciliation logic in
// the follow-up branch). See docs/ROADMAP.md for sequencing.
package v1

// ApplicationSpec is the desired state. Field shape matches vibeguard.yaml.
type ApplicationSpec struct {
	APIVersion string                 `json:"apiVersion,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Global     map[string]interface{} `json:"global,omitempty"`
	Modules    []map[string]interface{} `json:"modules,omitempty"`
	Platform   PlatformSpec           `json:"platform,omitempty"`
	Deployment DeploymentSpec         `json:"deployment,omitempty"`
}

// PlatformSpec configures the runtime drivers.
type PlatformSpec struct {
	DB     DriverSpec `json:"db,omitempty"`
	Events DriverSpec `json:"events,omitempty"`
	Cache  DriverSpec `json:"cache,omitempty"`
}

// DriverSpec selects a driver and points at its credentials.
type DriverSpec struct {
	Driver    string                 `json:"driver,omitempty"`
	URL       string                 `json:"url,omitempty"`
	SecretRef *SecretKeyRef          `json:"secretRef,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// SecretKeyRef references a key in a Kubernetes Secret.
type SecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// DeploymentSpec controls how the operator materializes the workload.
type DeploymentSpec struct {
	Image    string `json:"image,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Replicas int32  `json:"replicas,omitempty"`
	// Mode selects between operator-owned Deployment ("operator", default) and
	// emitting an argoproj.io/Application pointing at a kustomize overlay
	// ("argocd").
	Mode string `json:"mode,omitempty"`
}

// ApplicationStatus is the observed state, owned by the operator.
type ApplicationStatus struct {
	ObservedGeneration int64               `json:"observedGeneration,omitempty"`
	Conditions         []ApplicationCondition `json:"conditions,omitempty"`
	AppliedMigration   int                 `json:"appliedMigration,omitempty"`
	Streams            []string            `json:"streams,omitempty"`
	DriftDetected      bool                `json:"driftDetected,omitempty"`
	DriftDetail        string              `json:"driftDetail,omitempty"`
}

// ApplicationCondition is one of the standard reconciliation conditions:
// DeclarationValid, MigrationsApplied, EventsConfigured, WorkloadReady, NoDrift.
type ApplicationCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"` // "True" | "False" | "Unknown"
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}
