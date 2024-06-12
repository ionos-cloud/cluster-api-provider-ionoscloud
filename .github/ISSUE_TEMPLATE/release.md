---
name: Release
about: Release checklist
labels: kind/release

---

## Release

Release vX.X.X

## Checklist

- [ ] Update metadata & clusterctl-settings.
- [ ] Update version in e2e config.
- [ ] Update docs (compatibility table; usage etc).
- [ ] Create release branch `release-X.Y`
- [ ] Create tag `vX.Y.Z`.
- [ ] Update the created draft release to include things like breaking changes or important notes.
- [ ] Update the draft release notes with generated release notes from `sigs.k8s.io/kubebuilder-release-tools/notes`
- [ ] Publish the release.
