## REMOVED Requirements

### Requirement: Deep same-scope imports are limited within the discovered module scope
**Reason**: Import behavior now lives in the dedicated `import-control` capability instead of the retired mixed `architecture-control` capability.
**Migration**: Use `import-control` for deep same-scope import requirements.

### Requirement: Out-of-scope imports are ignored
**Reason**: Import-scope behavior now lives in the dedicated `import-control` capability instead of the retired mixed `architecture-control` capability.
**Migration**: Use `import-control` for out-of-scope import requirements.

### Requirement: Boundarycontrol uses selector-owned package policy
**Reason**: General selector ownership behavior now lives in the new `boundarycontrol` capability.
**Migration**: Use `boundarycontrol` for selector-owned package policy requirements.

### Requirement: Boundarycontrol resolves overlapping keys by nearest owner precedence
**Reason**: General selector precedence behavior now lives in the new `boundarycontrol` capability.
**Migration**: Use `boundarycontrol` for selector precedence requirements.

### Requirement: Boundarycontrol forbids cross-package imports unless explicitly allowed
**Reason**: Import-allowlist behavior now lives in the dedicated `import-control` capability.
**Migration**: Use `import-control` for explicit import policy requirements.

### Requirement: Architecture-control discovers module scope from go.mod
**Reason**: Module discovery is now part of the general `boundarycontrol` capability.
**Migration**: Use `boundarycontrol` for module discovery requirements.

### Requirement: Boundarycontrol import selectors use non-recursive package matching
**Reason**: Import selector matching now lives in the dedicated `import-control` capability.
**Migration**: Use `import-control` for import selector matching requirements.
