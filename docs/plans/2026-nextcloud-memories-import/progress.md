# Nextcloud Memories Import Progress

## Current Status

**Phase**: Proposal and UX draft

**Last Updated**: 2026-06-01

**Summary**:

The command shape, scope model, guardrails, and draft user-facing documentation have been outlined. No production code has been added yet. Public upload docs remain unchanged until the command exists.

---

## Step Tracking

- [x] Define the command direction and scope model
  - Import should target the configured Memories library, not arbitrary Nextcloud folders
  - Auto-discovery is the default behavior

- [x] Define escape hatches and guardrails
  - Drafted root-limiting, discovery-only, and indexing override options
  - Drafted failure behavior for missing app and invalid configuration

- [x] Draft user-facing command documentation
  - Added a planned command reference and examples in `user-docs-draft.md`
  - Kept public docs unchanged because the command is not implemented

- [ ] Implement source authentication and discovery

- [ ] Implement file enumeration from configured Memories roots

- [ ] Implement metadata and album mapping

- [ ] Add tests

- [ ] Promote draft docs into public docs after implementation ships

---

## Decisions

### 2026-06-01: Public Docs Stay Accurate

**Decision**: Do not add `from-nextcloud-memories` to `docs/commands/upload.md` yet.

**Rationale**:

- The command is not implemented
- The project guidelines require user-facing docs to track shipped behavior
- Draft UX docs belong in `docs/plans/` until code and tests exist

### 2026-06-01: No Positional Source Path

**Decision**: The proposed command should not take a positional `<source-path>`.

**Rationale**:

- A Memories migration should import the configured Memories library
- Requiring manual folder selection would make this a generic Nextcloud importer instead
- Existing `from-folder` already covers the generic import case

### 2026-06-01: `timeline_path` Defines Scope

**Decision**: Default import scope should be derived from the effective Memories `timeline_path` configuration.

**Rationale**:

- `timeline_path` is the actual Memories library scope
- `folders_path` is a UI navigation root, not the library definition
- Importing outside `timeline_path` would violate user expectations