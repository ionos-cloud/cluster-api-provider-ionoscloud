---
name: Release
about: Release checklist
labels: kind/release

---

## Release

Release vX.X.X

## Checklist

- [ ] (Minor Release) Update metadata & clusterctl-settings.
- [ ] (Minor Release) Update version in e2e config.
- [ ] Update docs (compatibility table; usage etc).
- [ ] (Minor Release) Create release branch `release-X.Y`
- [ ] (Patch Release) Cherry-pick all relevant changes into the release branch.
- [ ] Create tag `vX.Y.Z`.
- [ ] Update the draft release notes with generated release notes from `sigs.k8s.io/cluster-api/hack/tools/release/notes`
- [ ] Update the created draft release to include things like breaking changes or important notes.
- [ ] Check that the release contains the relevant artifacts.
- [ ] Publish the release.
- [ ] Test provider installation/upgrade with clusterctl
