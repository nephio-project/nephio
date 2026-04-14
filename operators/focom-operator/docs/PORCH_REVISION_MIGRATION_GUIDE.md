# Porch Revision Management: Migration from Nephio R4 to Post-R6

## Purpose

This document explains why `CreateDraftFromRevision` broke when upgrading from Nephio R4 to the tip of master (post-R6), what changed in Porch, and how we fixed it.

## Context

The FOCOM operator uses Nephio Porch as its storage backend. Porch manages PackageRevisions in Git repositories via the Kubernetes API. A core workflow is creating a new draft from an existing published revision (e.g., creating v2 from v1).

**Key file:** `focom-operator/internal/nbi/storage/porch.go`

---

## 1. The Old Way (Nephio R4)

### How It Worked

`CreateDraftFromRevision` delegated to `CreateDraft`, which created a blank PackageRevision and then manually wrote resources into it:

```
CreateDraftFromRevision(id, revisionID)
  │
  ├─ 1. Check no draft already exists
  ├─ 2. GetRevision(id, revisionID) → fetch source resource data
  └─ 3. CreateDraft(resourceType, revisionData)
         │
         ├─ POST PackageRevision (no tasks, no metadata.name)
         │     spec:
         │       packageName: "demo-ocloud-01"
         │       repository: "focom-resources"
         │       lifecycle: "Draft"
         │       workspaceName: "draft-demo-ocloud-01-1773402352"
         │
         ├─ Wait for PackageRevision to appear
         ├─ GET PackageRevisionResources
         ├─ PUT resources into draft (Kptfile + resource YAML)
         │     (retry loop, up to 10 attempts)
         └─ Done (~40s total)
```

### What Porch R4 Actually Recorded

When we inspect a draft created this way via kubectl, the tasks array looks like this:

```json
{
  "tasks": [
    { "type": "",     "init": { "description": "demo-ocloud-01 description" } },
    { "type": "eval", "eval": { "image": "render" } },
    { "type": "patch", "patch": { "patches": [ ... ] } },
    { "type": "eval", "eval": { "image": "render" } }
  ],
  "lifecycle": "Draft",
  "revision": null
}
```

Note: the first task has `"type": ""` (empty string), not `"type": "init"`. Porch R4 was lenient — it accepted an empty type with an `init` body and treated it as package initialization. The subsequent `eval` → `patch` → `eval` tasks are Porch recording the mutations that happened when our code wrote resources via PUT.

### Key Characteristics

- No `metadata.name` set (Porch auto-generated it as a hash: `focom-resources-fef5522...`)
- No `spec.revision` set (null until approval)
- No `spec.tasks` in the request (Porch added the implicit init task)
- Resources were copied manually via GET source → PUT into draft
- Porch R4 allowed creating new drafts for existing packages without explicit intent

---

## 2. What Broke (Post-R6)

The same code produced this error:

```
Internal error occurred: package "demo-ocloud-01" already exists in repository "focom-resources"
```

### Root Cause

Porch post-R6 changed how it interprets PackageRevision creation requests:

```
┌─────────────────────────────────────────────────────────────────────┐
│                    PORCH BEHAVIOR CHANGE                            │
├──────────────────────┬──────────────────────────────────────────────┤
│                      │                                              │
│   Request has no     │  R4: "OK, I'll create a new draft workspace │
│   tasks (or empty    │       for this existing package"             │
│   type init)         │                                              │
│                      │  Post-R6: "You're trying to INITIALIZE a    │
│                      │           new package, but it already        │
│                      │           exists → REJECTED"                 │
│                      │                                              │
├──────────────────────┼──────────────────────────────────────────────┤
│                      │                                              │
│   Request has        │  R4: N/A (not required)                      │
│   explicit edit      │                                              │
│   task with          │  Post-R6: "You want to create a new draft   │
│   sourceRef          │           from an existing revision → OK"    │
│                      │                                              │
└──────────────────────┴──────────────────────────────────────────────┘
```

In short: Porch post-R6 requires you to explicitly declare your intent when creating a draft for an existing package. The implicit "just create a blank workspace" approach no longer works.

---

## 3. Finding the Right Task Type

We tested every plausible task type before finding the one that works:

```
 Task Type Tested                          Result
 ─────────────────────────────────────────────────────────────────
 No tasks (old way)                        ❌ "package already exists in repository"
 clone                                     ❌ "clone cannot create a new revision for
                                              package that already exists; make
                                              subsequent revisions using copy"
 copy with source                          ❌ "task of type copy not supported"
 copy with upstream                        ❌ "task of type copy not supported"
 edit with upstream                        ❌ "unknown field spec.tasks[0].edit.upstream"
 edit with source (string)                 ❌ Schema error
 ─────────────────────────────────────────────────────────────────
 edit with sourceRef                       ✅ Works
```

---

## 4. The New Way (Post-R6) — The `edit` Task

### How It Works Now

```
CreateDraftFromRevision(id, revisionID)
  │
  ├─ 1. Check no draft already exists
  │
  ├─ 2. getPackageRevisionName(id, revisionID)
  │     └─ Returns full resource name of the source revision:
  │        "focom-resources.demo-ocloud-01.draft-...-1773411267"
  │
  ├─ 3. generateNextRevisionID(id)
  │     └─ Queries all Published revisions, finds highest number,
  │        returns next: "v2"
  │
  ├─ 4. POST PackageRevision with edit task:
  │       metadata:
  │         name: "focom-resources.demo-ocloud-01.v2-draft"
  │         namespace: "default"
  │       spec:
  │         packageName: "demo-ocloud-01"
  │         repository: "focom-resources"
  │         revision: 2              ← integer, not string
  │         lifecycle: "Draft"
  │         workspaceName: "v2-draft"
  │         tasks:
  │           - type: "edit"
  │             edit:
  │               sourceRef:
  │                 name: "focom-resources.demo-ocloud-01.draft-..."
  │
  ├─ 5. Wait for PackageRevision to appear
  │
  └─ 6. Done (~30s, no manual resource copying needed)
```

Porch copies all resources from the source revision automatically via the edit task.

### The Correct Request Body

```json
{
  "apiVersion": "porch.kpt.dev/v1alpha1",
  "kind": "PackageRevision",
  "metadata": {
    "namespace": "default",
    "name": "focom-resources.demo-ocloud-01.v2-draft"
  },
  "spec": {
    "packageName": "demo-ocloud-01",
    "repository": "focom-resources",
    "revision": 2,
    "workspaceName": "v2-draft",
    "lifecycle": "Draft",
    "tasks": [
      {
        "type": "edit",
        "edit": {
          "sourceRef": {
            "name": "focom-resources.demo-ocloud-01.draft-demo-ocloud-01-1773411267"
          }
        }
      }
    ]
  }
}
```

---

## 5. Side-by-Side Comparison

```
┌──────────────────────┬──────────────────────────────┬──────────────────────────────┐
│                      │  OLD WAY (R4)                │  NEW WAY (Post-R6)           │
├──────────────────────┼──────────────────────────────┼──────────────────────────────┤
│ metadata.name        │ Not set (auto-generated      │ Explicitly set:              │
│                      │ as hash)                     │ {repo}.{pkg}.{workspace}     │
├──────────────────────┼──────────────────────────────┼──────────────────────────────┤
│ spec.revision        │ Not set (null until          │ Integer (e.g., 2)            │
│                      │ approval)                    │                              │
├──────────────────────┼──────────────────────────────┼──────────────────────────────┤
│ spec.tasks           │ None in request              │ edit task with sourceRef     │
│                      │ (Porch added implicit        │ pointing to source           │
│                      │ init with type: "")          │ PackageRevision              │
├──────────────────────┼──────────────────────────────┼──────────────────────────────┤
│ spec.workspaceName   │ "draft-{id}-{timestamp}"     │ "{nextRev}-draft"            │
│                      │                              │ (e.g., "v2-draft")           │
├──────────────────────┼──────────────────────────────┼──────────────────────────────┤
│ Resource copying     │ Manual: GET source PRR →     │ Automatic: Porch copies      │
│                      │ PUT into draft (retry loop)  │ via edit task                │
├──────────────────────┼──────────────────────────────┼──────────────────────────────┤
│ Time                 │ ~40s (includes retry loop)   │ ~30s (no manual copying)     │
├──────────────────────┼──────────────────────────────┼──────────────────────────────┤
│ Intent declaration   │ Implicit (Porch guessed)     │ Explicit (edit task tells    │
│                      │                              │ Porch exactly what to do)    │
├──────────────────────┼──────────────────────────────┼──────────────────────────────┤
│ Works on Post-R6?    │ ❌ No                        │ ✅ Yes                       │
└──────────────────────┴──────────────────────────────┴──────────────────────────────┘
```

---

## 6. Why the Old Way Fails — A Visual

```
                         OLD WAY on Post-R6 Porch
                         ════════════════════════

  FOCOM Operator                              Porch (Post-R6)
  ──────────────                              ────────────────

  POST PackageRevision
    packageName: "demo-ocloud-01"
    tasks: (none)
                          ──────────►
                                              "No tasks? This must be a
                                               new package initialization."

                                              "But demo-ocloud-01 already
                                               exists in focom-resources..."

                          ◄──────────
                                              ❌ 500: "package demo-ocloud-01
                                                  already exists in repository
                                                  focom-resources"


                         NEW WAY on Post-R6 Porch
                         ════════════════════════

  FOCOM Operator                              Porch (Post-R6)
  ──────────────                              ────────────────

  POST PackageRevision
    packageName: "demo-ocloud-01"
    tasks:
      - type: "edit"
        edit:
          sourceRef:
            name: "focom-resources.
              demo-ocloud-01.draft-..."
                          ──────────►
                                              "Edit task with sourceRef?
                                               This is a new draft derived
                                               from an existing revision."

                                              "Creating draft, copying
                                               resources from source..."

                          ◄──────────
                                              ✅ 201 Created
```

---

## 7. Additional Fixes Made During This Work

### 7.1 Revision Number Matching (`getPackageRevisionName`)

The `getPackageRevisionName` helper was comparing `spec.revision` as a string, but Porch post-R6 stores it as an integer (JSON number). Fixed to handle both:

```go
// Before (broken on post-R6):
revision, _ := spec["revision"].(string)  // Always empty — revision is int/float64

// After (works on both):
var revNum int
switch v := spec["revision"].(type) {
case int:      revNum = v
case float64:  revNum = int(v)
case string:   fmt.Sscanf(v, "v%d", &revNum)  // Legacy fallback
}
```

The same fix was applied in `generateNextRevisionID`.

### 7.2 FPR ID Generation (Separate Issue)

FocomProvisioningRequest was generating IDs by concatenating `oCloudID + templateName + templateVersion`. Changed to use just the `name` field, matching the OCloud pattern. See `.kiro/specs/focom-operator/fix-fpr-id-generation/`.

---

## 8. Files Changed

| File | Change |
|------|--------|
| `focom-operator/internal/nbi/storage/porch.go` | Rewrote `CreateDraftFromRevision` to use `edit` task with `sourceRef`; added `getPackageRevisionName` and `generateNextRevisionID` helpers; fixed revision number type handling |

---

## 9. How to Verify

### Inspect Existing PackageRevisions

```bash
kubectl get packagerevisions -n default -o json | jq '.items[] | select(.spec.packageName == "demo-ocloud-01") | {name: .metadata.name, tasks: .spec.tasks, lifecycle: .spec.lifecycle, revision: .spec.revision}'
```

What to look for:
- Old-way drafts: first task has `"type": ""` with an `init` body, followed by `eval`/`patch`/`eval` tasks from manual resource writing
- New-way drafts: single `"type": "edit"` task with `sourceRef` pointing to the source revision

### Manual Test (Postman)

1. Create an OCloud and approve it (creates v1)
2. Use "Create OCloud Draft from Revision" with revisionId = "v1"
3. Modify, validate, approve the draft (creates v2)
4. Verify both v1 and v2 exist via "Get OCloud Revisions"

### Integration Tests

```bash
cd focom-operator
go test -v ./internal/nbi -run TestOCloudRevisionManagement
go test -v ./internal/nbi -run TestTemplateInfoRevisionManagement
go test -v ./internal/nbi -run TestFocomProvisioningRequestRevisionManagement
```

### Direct kubectl (Bypass FOCOM Operator)

```bash
cat <<EOF | kubectl create -f -
{
  "apiVersion": "porch.kpt.dev/v1alpha1",
  "kind": "PackageRevision",
  "metadata": {
    "namespace": "default",
    "name": "focom-resources.demo-ocloud-01.v2-draft"
  },
  "spec": {
    "packageName": "demo-ocloud-01",
    "repository": "focom-resources",
    "revision": 2,
    "workspaceName": "v2-draft",
    "lifecycle": "Draft",
    "tasks": [
      {
        "type": "edit",
        "edit": {
          "sourceRef": {
            "name": "focom-resources.demo-ocloud-01.REPLACE-WITH-SOURCE-WORKSPACE"
          }
        }
      }
    ]
  }
}
EOF
```

---

## 10. Open Items

- Integration tests have not been run against the new code yet (Task 5 in tasks.md)
- Debug `fmt.Printf` statements in `CreateDraftFromRevision` should be removed before merging
