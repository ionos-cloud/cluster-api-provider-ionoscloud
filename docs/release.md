# Release

> [!NOTE]
> This document is based on the initial learnings from the release process and is therefore subject to change.

## Preparing the release

For each release, a milestone needs to be in place. All PRs that are part of the release need to be assigned to 
the milestone.

This will also make it easier to filter for the correct commits, that should be cherry-picked later.

### Create an issue

To track the progress of the release, we need to create an issue. Make use of the `Release` issue template.

### Patch release

A patch release can only include bug fixes, documentation updates, and other minor changes.

New features and breaking changes must be part of a minor release.

### Minor release

A minor release includes all changes from the last minor version to the next one. 
This also includes new features and breaking changes.

The following tasks need to be done before creating a new release:

- Update metadata & clusterctl-settings.
- Update version in e2e config.
- Update docs (compatibility table; usage etc).

### Milestones

With `Milestones` we want to make it easier to track the progress of a release.

### Cherry-Picking

> [!NOTE]
> As we do not yet have `Prow`, we need to do the cherry-picking manually.
> Refer to the [git cherry-pick docs](https://git-scm.com/docs/git-cherry-pick)

#### Patch release

For each release, we need to cherry-pick all the relevant changes. For patch releases, we already
have a release branch in place. All changes, that should go into the next patch version, need to be merged into
the corresponding release branch. 

After this is done, we need to create a new tag for the release.

#### Minor release

A minor release requires the creation of a new release branch. We will include all changes from the mainline
into the release branch. This includes new features and breaking changes.

In this case we do not have to `cherry-pick` the changes.

### Generate release notes

Cluster API provides a tool to generate release notes. The tool can be found in the `hack/tools/release/notes` directory
within the Cluster API repository.

An example for creating release notes with the tool could be:
```bash
# make sure you are in the cluster-api repository
cd ../cluster-api
# run the release notes tool and provide correct Github repository
go run -tags=tools sigs.k8s.io/cluster-api/hack/tools/release/notes \
  --from tags/v0.Y.Z \
  --to tags/v0.X.0 \
  --branch release-0.X \
  --repository ionos-cloud/cluster-api-provider-ionoscloud
```

The terminal output can then be copied into the draft release notes. There might be some manual work left to do.

