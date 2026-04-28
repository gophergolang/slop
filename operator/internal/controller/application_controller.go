// Package controller is the Application reconciler.
//
// Status: scaffold only on branch 4-7. The follow-up branch wires
// controller-runtime (sigs.k8s.io/controller-runtime), implements the five
// reconciliation conditions (DeclarationValid, MigrationsApplied,
// EventsConfigured, WorkloadReady, NoDrift), and adds the validating
// admission webhook + Helm chart.
//
// The reconciliation contract is documented in docs/ARCHITECTURE.md §5
// (Pillar 3 — Operator). Reconcilers receive an Application CR, parse the
// spec via internal/decl (the same parser the CLI uses), and own:
//
//   1. Migrations    — golang-migrate against the declared database
//   2. NATS streams  — JetStream CreateOrUpdateStream/Consumer
//   3. Workload      — Deployment + Service + NetworkPolicy + HPA + PDB
//   4. Secrets       — ExternalSecret refs to ClusterSecretStore
//   5. RLS drift     — pg_policies introspection vs. declaration
package controller

// Reconcile is the single entry-point invoked by controller-runtime per
// Application change. The full implementation lands in branch 4-7-followup.
func Reconcile() error { return nil }
