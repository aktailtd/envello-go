# Update repo URLs to aktailtd

**Goal:** the repo moved from Bergblumestudio to the `aktailtd` org; point metadata at the new home.

**What changed:** README.md go.mod : replaced `github.com/Bergblumestudio/` with `github.com/aktailtd/`.

**Decisions:** URL-only change, no version bump; publishing a release is left to the owner.

**State:** done. Same change opened on all six envello SDK repos.

**Verify:** `grep -ri bergblume .` returns nothing. `go test ./...` passes.

**Gotchas:**
- GitHub redirects old URLs only until a repo is created at the old path.
- The module path changed, so importers on the old path get a mismatch error. Tag a new release after merge so `go get github.com/aktailtd/envello-go@latest` resolves.
