# Refactor Plan: Move Commands from `svc` to Root

## Overview
Moving the following commands from `svc` to root level:
- `svc open` → `open` (root)
- `svc variables` → `vars` (root, renamed)
- `svc pipeline` → `pipelines` (root, renamed)
- `svc ecr` → `ecr` (root)
- `svc ingress` → `ingress` (root)

Commands staying under `svc`:
- `svc list`
- `svc new`
- `svc status`

## Step-by-Step Plan

### Phase 1: Move `open` Command to Root
**File:** `cmd/svc_open.go`

1. ✅ Change command registration:
   - Line 158: Change `svcCmd.AddCommand(svcOpenCmd)` → `rootCmd.AddCommand(svcOpenCmd)`

2. ✅ Update command Use field:
   - Line 13: Already correct `Use: "open [service-name]"`

3. ✅ Update error messages:
   - Line 23: Update usage hint in Long description if needed (optional)

4. ✅ Subcommands stay the same:
   - `pipelinesCmd`, `prsCmd`, `varsCmd`, `settingsCmd` remain subcommands of `open`

**No file rename needed** (keeping `svc_open.go` is fine)

---

### Phase 2: Move `variables` → `vars` Command to Root
**File:** `cmd/svc_variables.go`

1. ✅ Change command registration:
   - Line 264: Change `svcCmd.AddCommand(svcVariablesCmd)` → `rootCmd.AddCommand(svcVariablesCmd)`

2. ✅ Rename command:
   - Line 23: Change `var svcVariablesCmd` → `var varsCmd`
   - Line 24: Change `Use: "variables [service-name]"` → `Use: "vars [service-name]"`
   - Line 25: Update Short/Long descriptions if needed

3. ✅ Update error messages:
   - Line 55: Change `"eiscli svc variables <service-name>"` → `"eiscli vars <service-name>"`

4. ✅ Update subcommand references:
   - Subcommands (`add`, `sync`) reference `svcVariablesCmd` → update to `varsCmd`

**Files affected:**
- `cmd/svc_variables.go` (main command)
- `cmd/svc_variables_add.go` (references parent command)
- `cmd/svc_variables_sync.go` (references parent command)

**In `svc_variables_add.go`:**
- Line 60: Change `"eiscli svc variables add <service-name>"` → `"eiscli vars add <service-name>"`
- Line 96: Change `"eiscli svc variables add [service-name]"` → `"eiscli vars add [service-name]"`
- Line 361: Change `svcVariablesCmd.AddCommand(...)` → `varsCmd.AddCommand(...)`

**In `svc_variables_sync.go`:**
- Line 77: Change `"eiscli svc variables sync <service-name>"` → `"eiscli vars sync <service-name>"`
- Line 87: Change `"eiscli svc variables sync [service-name]"` → `"eiscli vars sync [service-name]"`
- Line 407: Change `svcVariablesCmd.AddCommand(...)` → `varsCmd.AddCommand(...)`

---

### Phase 3: Move `pipeline` → `pipelines` Command to Root
**File:** `cmd/svc_pipeline.go`

1. ✅ Change command registration:
   - Line 223: Change `svcCmd.AddCommand(svcPipelineCmd)` → `rootCmd.AddCommand(pipelinesCmd)`

2. ✅ Rename command:
   - Line 20: Change `var svcPipelineCmd` → `var pipelinesCmd`
   - Line 21: Change `Use: "pipeline [service-name]"` → `Use: "pipelines [service-name]"`
   - Line 22: Update Short/Long descriptions

3. ✅ Update error messages:
   - Line 45: Change `"eiscli svc pipeline <service-name>"` → `"eiscli pipelines <service-name>"`

4. ✅ Update flag references:
   - Lines 224-226: Change `svcPipelineCmd.Flags()` → `pipelinesCmd.Flags()`

---

### Phase 4: Move `ecr` Command to Root
**File:** `cmd/svc_ecr.go`

1. ✅ Change command registration:
   - Line 198: Change `svcCmd.AddCommand(svcECRCmd)` → `rootCmd.AddCommand(svcECRCmd)`

2. ✅ Update command Use field:
   - Line 23: Already correct `Use: "ecr [service-name]"`

3. ✅ Update error messages:
   - Line 137: Change `"eiscli svc ecr %s --region %s --create"` → `"eiscli ecr %s --region %s --create"`
   - Line 166: Change `"eiscli svc ecr %s --region %s --create"` → `"eiscli ecr %s --region %s --create"`

**File:** `cmd/svc_ecr_update_workspace_var.go`

4. ✅ Update error messages:
   - Line 36: Change `'eiscli svc ecr --create'` → `'eiscli ecr --create'`
   - Line 93: Change `"eiscli svc ecr update-workspace-var <service-name>"` → `"eiscli ecr update-workspace-var <service-name>"`
   - Line 131: Change `"eiscli svc ecr %s --region %s --create"` → `"eiscli ecr %s --region %s --create"`

5. ✅ Subcommand stays:
   - `update-workspace-var` remains subcommand of `ecr`

---

### Phase 5: Move `ingress` Command to Root
**File:** `cmd/svc_ingress.go`

1. ✅ Change command registration:
   - Line 102: Change `svcCmd.AddCommand(svcIngressCmd)` → `rootCmd.AddCommand(svcIngressCmd)`

2. ✅ Update command Use field:
   - Line 85: Already correct `Use: "ingress"`

3. ✅ Subcommand stays:
   - `add` remains subcommand of `ingress`

**No error message updates needed** (no hardcoded references found)

---

### Phase 6: Update Helper Functions
**File:** `cmd/helpers.go`

1. ✅ Update `getServiceName()` error message:
   - Line 23: Change `"eiscli svc <command> <service-name>"` → `"eiscli <command> <service-name>"`

**Note:** This is a generic helper, so the generic message is fine, but we could make it more specific if needed.

---

### Phase 7: Update Remaining `svc` Commands (Error Messages)
**Files:** Commands staying under `svc`

1. ✅ `cmd/svc_status.go`:
   - Line 36: Change `"eiscli svc status <service-name>"` → keep as is (correct)

2. ✅ `cmd/svc_new.go`:
   - Line 23: Change `"eiscli svc new <service-name>"` → keep as is (correct)

**These are correct as-is since they're staying under `svc`**

---

## Summary of Changes

### Command Registrations (Change `svcCmd` → `rootCmd`):
- ✅ `cmd/svc_open.go` - Line 158
- ✅ `cmd/svc_variables.go` - Line 264
- ✅ `cmd/svc_pipeline.go` - Line 223
- ✅ `cmd/svc_ecr.go` - Line 198
- ✅ `cmd/svc_ingress.go` - Line 102

### Command Renames:
- ✅ `svcVariablesCmd` → `varsCmd` (variable name)
- ✅ `svcPipelineCmd` → `pipelinesCmd` (variable name)
- ✅ `Use: "variables"` → `Use: "vars"`
- ✅ `Use: "pipeline"` → `Use: "pipelines"`

### Error Message Updates:
- ✅ `cmd/helpers.go` - Line 23
- ✅ `cmd/svc_variables.go` - Line 55
- ✅ `cmd/svc_variables_add.go` - Lines 60, 96
- ✅ `cmd/svc_variables_sync.go` - Lines 77, 87
- ✅ `cmd/svc_pipeline.go` - Line 45
- ✅ `cmd/svc_ecr.go` - Lines 137, 166
- ✅ `cmd/svc_ecr_update_workspace_var.go` - Lines 36, 93, 131

### Subcommand References:
- ✅ `cmd/svc_variables_add.go` - Line 361: `svcVariablesCmd` → `varsCmd`
- ✅ `cmd/svc_variables_sync.go` - Line 407: `svcVariablesCmd` → `varsCmd`
- ✅ `cmd/svc_pipeline.go` - Lines 224-226: `svcPipelineCmd` → `pipelinesCmd`

## Testing Checklist

After refactoring, test each command:

- [ ] `eiscli open <service>` and subcommands (`pipelines`, `prs`, `vars`, `settings`)
- [ ] `eiscli vars <service>` and subcommands (`add`, `sync`)
- [ ] `eiscli pipelines <service>`
- [ ] `eiscli ecr <service>` and subcommand (`update-workspace-var`)
- [ ] `eiscli ingress add`
- [ ] `eiscli svc list` (should still work)
- [ ] `eiscli svc new <service>` (should still work)
- [ ] `eiscli svc status <service>` (should still work)
- [ ] `eiscli --help` (verify structure)
- [ ] Error messages display correct usage

## Optional: File Renaming (Low Priority)

Consider renaming files for consistency (optional):
- `svc_open.go` → `open.go`
- `svc_variables.go` → `vars.go`
- `svc_variables_add.go` → `vars_add.go`
- `svc_variables_sync.go` → `vars_sync.go`
- `svc_pipeline.go` → `pipelines.go`
- `svc_ecr.go` → `ecr.go`
- `svc_ecr_update_workspace_var.go` → `ecr_update_workspace_var.go`
- `svc_ingress.go` → `ingress.go`

**Note:** File renaming is optional and doesn't affect functionality.

