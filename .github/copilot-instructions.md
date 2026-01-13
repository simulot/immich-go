# GitHub Copilot Instructions

## immich-go — Solo Maintainer, Low-Process Mode

These instructions define how GitHub Copilot should assist with the maintenance and evolution of **immich-go**, a Go-based open-source CLI project, maintained by a single developer.

The goal is **code quality, safety, and long-term maintainability**, without introducing heavy project management overhead.

---

## 1. Go Programming Style

Copilot must follow **idiomatic Go practices**, inspired by official Go guidelines and the GitHub Copilot Go instructions:

* Prefer **clarity over cleverness**
* Small, composable functions
* Explicit error handling (no hidden control flow)
* Meaningful variable and function names
* Avoid unnecessary abstractions
* Keep public APIs minimal and intentional
* Follow `gofmt`, `go vet`, and `golangci-lint` expectations
* Respect existing project conventions and patterns

Do **not** introduce patterns that feel foreign to standard Go projects.

---

## 2. Tests Are Mandatory

* Any **new feature**, **bug fix**, or **behavioral change** must include tests.
* If existing code lacks coverage in a modified area:

  * Propose missing tests
  * Or explain clearly why tests are impractical
* Tests must be:

  * Readable
  * Deterministic
  * Focused on behavior, not implementation details

Copilot should **actively suggest tests** when they are missing.

---

## 3. Documentation Requirements

* Generate **GoDoc comments** for:

  * New public functions, types, and packages
  * Modified legacy code when touched
* Improve clarity of existing comments when refactoring
* Documentation must explain **intent**, not restate the code

No undocumented public API is acceptable.

---

## 4. Dependency Discipline

* Prefer the **Go standard library**
* Introduce external dependencies only when:

  * There is a strong, explicit justification
  * The benefit clearly outweighs the maintenance cost
* Always challenge dependency additions

---

## 5. Planning: When, Where, and How

### 5.1 When a Plan Is Required

A plan is required when:

* The change is **non-trivial**
* The change affects multiple packages or layers
* The change introduces refactoring or technical debt reduction
* The change may impact users (CLI, behavior, data)

Small, obvious fixes do **not** require a plan.

---

### 5.2 Where Plans Live (Acceptable Place)

All planning, design notes, and tracking documents must live in:

```
docs/plans/
```

Structure example:

```
docs/
└── plans/
    ├── 2025-immich-go-refactor/
    │   ├── README.md        # high-level goal and scope
    │   ├── plan.md          # step-by-step plan
    │   └── progress.md      # current status and notes
```

This keeps:

* Design decisions close to the code
* History explicit
* No dependency on external tools

---

### 5.3 How to Write a Plan (Solo Maintainer Style)

Plans must be:

* Short
* Concrete
* Step-based

Each plan should include:

1. **Goal** (what problem is being solved)
2. **Non-goals** (what is explicitly out of scope)
3. **Steps**, each:

   * Small
   * Independently mergeable
   * Testable

Avoid vague or ambitious steps.

---

### 5.4 Tracking Progress

Tracking is lightweight:

* Use a simple checklist or numbered steps in `progress.md`
* Update it only when:

  * A step is completed
  * A decision changes
* No dashboards, no automation

Copilot may suggest updating progress when a step is completed.

---

## 6. Refactoring Legacy Code

Refactoring must be:

* **Planned**
* **Incremental**
* **Safe**

Rules:

* Never mix refactoring with new features
* Reduce technical debt step by step
* Each refactor must:

  * Preserve behavior
  * Improve readability, testability, or structure
  * Include tests if missing

Large refactors must be broken down into **multiple small PRs** that can be merged independently to keep `main` / `develop` divergence minimal.

Copilot should challenge any refactor that is too broad.

---

## 7. Scope Control & Clarification

Copilot must:

* Ask for clarification when requirements are ambiguous
* Challenge unrealistic or overly broad requests
* Propose breaking large goals into manageable steps
* Prefer several small changes over one large change

---

## 8. Commits & Pull Requests

### 8.1 Conventional Commits

Use **Conventional Commits**, with:

* Short, imperative messages
* Focus on the key change

Examples:

* `feat(cli): add dry-run mode to upload`
* `fix(upload): handle empty album names`
* `refactor(ui): simplify terminal rendering logic`

### 8.2 User-Facing Changes

Commit messages and PR descriptions must clearly mention:

* CLI changes
* New features
* Behavior changes that affect users

### 8.3 Pull Request Descriptions

PR descriptions must be:

* Lean
* Informative
* Structured

Include:

* What changed
* Why
* User impact (if any)
* Reference to plan step (if applicable)

---

## 9. Default Assumption

This is a **solo-maintained project**.

Copilot should:

* Optimize for maintainability, not coordination
* Avoid suggesting heavy workflows (projects, epics, ceremonies)
* Respect the maintainer’s limited cognitive overhead

Process exists **to serve the code**, not the opposite.

---

## 10. Documentation Maintenance Before Merging

Before merging any feature, refactor, or behavior change, Copilot must ensure that **project documentation remains accurate and consistent with the code**.

Documentation is considered **part of the deliverable**, not an afterthought.

---

### 10.1 Documents That Must Be Reviewed

For any non-trivial change, review and update when applicable:

* `README.md` (root or submodules)
  * CLI usage
  * Examples
  * Installation or configuration changes


* Architecture or design documents
  * Data flow
  * Package responsibilities
  * Key abstractions

* User-facing documentation
  * CLI flags
  * Behavior changes
  * Breaking or deprecated features

* Planning documents in `docs/plans/`
  * Mark steps as completed
  * Update decisions if they changed

If documentation is impacted but **not updated**, Copilot must flag it explicitly.

---

### 10.2 Architecture Consistency

Copilot must:

* Detect when a change affects project architecture
* Prompt for updating architecture documentation
* Keep diagrams, explanations, and structure aligned with actual code behavior

Avoid architecture drift between documentation and implementation.

---

### 10.3 Definition of “Ready to Merge”

A change is considered **ready to merge** only when:

* Code is implemented and reviewed
* Tests are present and passing
* GoDoc is updated
* User-facing changes are documented
* Architecture and design docs reflect the new reality
* Planning documents are updated (if applicable)

Copilot should **block or warn** when any of these are missing.

---

### 10.4 Scope Awareness

Documentation updates must:

* Match the scope of the change
* Stay concise
* Avoid speculative or future-looking content

Only document what is **implemented and merged**.

---

### 10.5 Solo Maintainer Constraint

Documentation work should:

* Be incremental
* Be done alongside the code change
* Avoid large, disruptive rewrites unless explicitly planned

Copilot should **prefer small doc updates per change**, not large documentation overhauls.

---

