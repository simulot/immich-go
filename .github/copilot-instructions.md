You are helping me for the design. don't update the code if not explicitly requested

Don't write reasoning documentation 
    except is this is explicitly requested. Write it in the scratchpad folder.

When implementing an immich end point, 
    refer to the api specifications located at .github/immich-api-monitor/immich-openapi-specs-baseline.json

Never commit the code by yourself.
   prepare the commit message and ask for confirmation before 

## Release Notes Generation

When asked to generate release notes:
1. Use the script: `./scripts/generate-release-notes.sh [version]`
   - Example: `./scripts/generate-release-notes.sh v0.30.0`
   - Without version, it defaults to "NEXT"

2. The script will:
   - Find commits since the last stable release
   - Generate a prompt file: `release-notes-prompt.txt`
   - if the file: `docs/releases/release-notes-[version].md` already exists, it will be used as a base
   - Create target file: `docs/releases/release-notes-[version].md`

3. Process the prompt with this chat to generate the final release notes

4. Release notes should be categorized as:
   - ✨ New Features (user-visible functionality)
   - 🚀 Improvements (enhancements to existing features)
   - 🐛 Bug Fixes (fixes to existing functionality)
   - 💥 Breaking Changes (changes requiring user action)
   - 🔧 Internal Changes (refactoring, CI/CD, tests - only if significant)

5. Guidelines:
   - Remove technical prefixes (feat:, fix:, chore:, refactor:, doc:, e2e:, test:)
   - Write from user's perspective
   - Combine related commits
   - Start with a brief, friendly introduction. Be concise and professional.
   - Explain CLI changes, list concerned flags, add examples if needed
   - Skip mean less commits (e.g., "update README", "fix typo")
   - Skip purely internal changes unless they significantly impact users
   

## commit messages generation
  - use conventional commit style
  - the commit title should prioritize features that affects the user experience
  - the commit details list other changes
  - maintain a provisional change log in the folder scratchpad
  - but newer commit scratchpad content
  - only changes in the command line options are concidered as breaking change. The project is not about publish an API or a library.

## prepare a pull-request to merge with the develop branch

- if the file: `docs/releases/release-notes-[version].md` already exists, it will be used as a base
- use the git commits messages
- use the provisional change log in the folder scratchpad if present 
- the pull-request should be named: `feature: [feature]`

## prepare a pull-request to merge with the main branch

- propose the new version using semantic versioning
- be sure that the file `docs/releases/release-notes-[version].md` exists or generate it.
- if the `docs/releases/release-notes-[NEXT].md` exists, it will be used as a base, and rename it with the new version number
- the pull-request should be named: `release: [version]`



## Development Conventions

-   **Branching:** The project follows a specific branching strategy.
    -   `feature/*` and `bugfix/*` branches should be based on and merged into `develop`.
    -   `hotfix/*` branches should be based on and merged into `main`.
-   **Commits:** While not strictly enforced, it's good practice to follow conventional commit message formats.
-   **Dependencies:** Manage dependencies using Go modules (`go.mod` and `go.sum`).
-   **Contributing:** Refer to `CONTRIBUTING.md` for more detailed contribution guidelines.


## Working on a new feature

- Storyboarding and design discussions should be documented in the `scratchpad/` directory.
- Implementation progress and summaries should also be documented in the `scratchpad/` directory.   
- propose unit tests for testing tricky aspects of the feature
- propose e2e tests:
    - explain the case to be tested: 
    - propose a description of test data needed for the test. 
    - I'll provide the data and files for the test
    - ensure that the entire upload process works as expected, including file discovery, processing, and uploading to the Immich server.
