# Veterinary Surgery Example

This example declares a multi-tenant veterinary surgery app for a small practice or clinic group. It models the operational core: staff access, clients, animal patients, appointments, clinical notes, prescriptions, and invoices.

The goal is to show how vibeguard can keep a regulated, data-sensitive workflow explicit and reviewable before any code is generated.

## Files

- `vibeguard.yaml` - the application declaration.

## Domain Shape

The declaration is split into three modules:

- `auth` - staff users and role-based access for practice owners, vets, nurses, receptionists, and finance users.
- `practice` - client records, pets, appointment scheduling, clinical notes, prescriptions, row-level security, and operational/audit events.
- `billing` - invoices tied to clients and appointments.

The main entity graph is:

```text
Client
  Pet
    Appointment
    ClinicalNote
    Prescription

Invoice -> Client
Invoice -> Appointment
```

All entities include `tenant_id` because the app is declared as row-isolated multi-tenant software.

## Security Posture

The example uses explicit CRUD whitelists and role restrictions rather than broad defaults:

- Receptionists can manage clients, pets, appointments, and invoices, but not clinical notes or prescriptions.
- Nurses can work with clients, pets, appointments, and clinical notes.
- Vets can access clinical records and prescriptions.
- Finance users are limited to invoice workflows.
- Deletions are disabled; patient and appointment records use retention-friendly update/status flows instead.

Clinical notes and prescriptions are marked `restricted` because they contain sensitive medical records.

## Custom Logic

The appointment check-in endpoint is declared as a node-backed endpoint:

```yaml
custom_endpoints:
  - path: /api/v1/appointments/:id/check-in
    method: POST
    node: practice.CheckInAppointment
```

The generated wrapper should own request parsing, auth, tenant context, transaction boundaries, and observability. The developer-owned node should only implement the business logic for checking in the appointment and deciding any side effects.

## Try It

From the repository root:

```bash
make build
./bin/vibeguard validate -f examples/veterinary-surgery/vibeguard.yaml
./bin/vibeguard ir dump -f examples/veterinary-surgery/vibeguard.yaml
./bin/vibeguard generate -f examples/veterinary-surgery/vibeguard.yaml -o /tmp/veterinary-surgery
```

To build the generated project:

```bash
cd /tmp/veterinary-surgery
echo 'replace github.com/vibeguard/platform => /path/to/slop/platform' >> go.mod
go mod tidy
go build ./...
```

Replace `/path/to/slop/platform` with the path to your local `platform` module.
