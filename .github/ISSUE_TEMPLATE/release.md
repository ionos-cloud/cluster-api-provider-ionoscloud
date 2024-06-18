---
name: Release
about: Release checklist
labels: kind/release

---

## Release

Release vX.X.X

## Checklist

- [ ] (optional) Update metadata & clusterctl-settings.
- [ ] (optional) Update version in e2e config.
- [ ] Update docs (compatibility table; usage etc).
- [ ] (optional) Create release branch `release-X.Y`
- [ ] Cherry-pick all relevant changes into the release branch.
- [ ] Create tag `vX.Y.Z`.
- [ ] Update the created draft release to include things like breaking changes or important notes.
- [ ] Update the draft release notes with generated release notes from `sigs.k8s.io/cluster-api/hack/tools/release/notes`
- [ ] Check that the release contains the relevant artifacts.
- [ ] Publish the release.
- [ ] Test provider installation/upgrade with clusterctl
