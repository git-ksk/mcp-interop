# Repository protection policy

`main` is protected by the repository Ruleset **Protect main**. This is a GitHub repository setting and therefore cannot be enforced by Git-tracked files alone.

The active policy for `refs/heads/main` is:

- branch deletion is blocked;
- non-fast-forward updates / force pushes are blocked;
- changes must arrive through a pull request;
- zero approving reviews are required so a solo maintainer is not deadlocked;
- review threads must be resolved;
- only squash merge is allowed;
- required status checks are `test (ubuntu-latest)`, `test (macos-latest)`, and `test (windows-latest)`;
- linear history is required;
- there is no configured bypass actor.

The privileged `real-client-e2e` Environment remains separately restricted to `main`. The workflow itself additionally verifies repository identity, exact `main` ref/workflow provenance, and the trusted self-hosted runner boundary.

Maintainers should re-audit this Ruleset before a stable release if repository settings are migrated, recreated, or transferred. A missing classic branch-protection response does not prove that `main` is unprotected; repository Rulesets must also be inspected.
