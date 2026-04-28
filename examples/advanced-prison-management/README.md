# Advanced Prison Management Example

This example declares a multi-tenant prison management system for secure custody operations. It is intentionally broad: it covers facilities, housing, inmate records, movements, incidents, evidence, rehabilitation programs, parole reviews, visits, healthcare records, staff roles, row-level security, and audit events.

The point of this example is to show how vibeguard can force a high-risk public-sector workflow into an explicit, reviewable declaration before code is generated.

## Files

- `vibeguard.yaml` - the application declaration.

## Domain Shape

The declaration is split into six modules:

- `auth` - staff users, badge numbers, active status, and role-based access.
- `custody` - facilities, housing units, inmates, custody classification, and inmate movements.
- `safety` - incident reports and contraband evidence with chain-of-custody data.
- `rehabilitation` - programs, enrollments, and parole review workflows.
- `visits` - visitor approvals and visit scheduling.
- `healthcare` - restricted medical records for inmates.

The main operational graph is:

```text
Facility
  HousingUnit
  Inmate
    Movement
    ProgramEnrollment
    ParoleReview
    Visit
    MedicalRecord

IncidentReport
  ContrabandEvidence

Visitor
  Visit
```

Every persisted entity includes `tenant_id` because the app is declared as row-isolated multi-tenant software. In a real deployment, a tenant could represent a prison estate, agency, region, or managed facility group.

## Security Posture

The declaration treats custody data as sensitive by default:

- Inmate, movement, incident, evidence, parole, visit, and medical records are `restricted` or `confidential`.
- Deletions are disabled across the system. Corrections happen through status changes and audit events.
- Warden and deputy warden roles own the highest-risk workflows.
- Intelligence officers can access investigations, incidents, evidence, visitor risk, and reclassification workflows.
- Healthcare staff can access medical records but do not receive blanket custody administration privileges.
- Auditors are granted read-oriented access through explicit API role lists.

The example uses row-level security policies for each module so records are constrained by `tenant_id`.

## Custom Logic

Several high-risk operations are declared as node-backed endpoints:

```yaml
custom_endpoints:
  - path: /api/v1/inmates/:id/reclassify
    method: POST
    node: custody.ReclassifyInmate

  - path: /api/v1/movements/:id/approve
    method: POST
    node: custody.ApproveMovement

  - path: /api/v1/incidents/:id/submit
    method: POST
    node: safety.SubmitIncident

  - path: /api/v1/visits/:id/approve
    method: POST
    node: visits.ApproveVisit
```

The generated wrapper should own request parsing, auth, tenant context, transactions, and observability. The developer-owned node should implement the domain decision, such as checking classification rules, visitor restrictions, or supervisor approval requirements.

## Event Model

The declaration emits audit or internal events for important custody transitions:

- `InmateAdmitted`
- `InmateReclassified`
- `MovementApproved`
- `MovementCompleted`
- `IncidentSubmitted`
- `EvidenceTransferred`
- `ProgramEnrollmentCompleted`
- `ParoleReviewSubmitted`
- `VisitApproved`
- `MedicalRecordLocked`

These are intentionally declared rather than implied, so reviewers can inspect exactly which state changes produce operational or audit signals.

## Try It

From the repository root:

```bash
make build
./bin/vibeguard validate -f examples/advanced-prison-management/vibeguard.yaml
./bin/vibeguard ir dump -f examples/advanced-prison-management/vibeguard.yaml
./bin/vibeguard generate -f examples/advanced-prison-management/vibeguard.yaml -o /tmp/advanced-prison-management
```

To build the generated project:

```bash
cd /tmp/advanced-prison-management
echo 'replace github.com/vibeguard/platform => /path/to/slop/platform' >> go.mod
go mod tidy
go build ./...
```

Replace `/path/to/slop/platform` with the path to your local `platform` module.
