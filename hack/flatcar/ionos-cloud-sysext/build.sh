#!/usr/bin/env bash
# Builds the ionos-cloud systemd-sysext (squashfs) from rootfs/ and sysext.yaml.
# Usage: build.sh [output-dir]   -> prints the path of ionos-cloud-<version>-<arch>.raw
set -euo pipefail

src=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
outdir=${1:-.}
meta="${src}/sysext.yaml"

die() { echo "build.sh: $*" >&2; exit 1; }
cfg() { sed -n "s/^$1: *\"\{0,1\}\([^\"]*\)\"\{0,1\} *$/\1/p" "${meta}"; }

command -v mksquashfs >/dev/null || die "mksquashfs not found (squashfs-tools)"

name=$(cfg name)
version=$(cfg version)
id=$(cfg id)
level=$(cfg sysext_level)
arch=$(cfg architecture)
[[ "${name}" =~ ^[a-z0-9][a-z0-9-]*$ ]] || die "invalid name '${name}' in ${meta}"
[ -n "${version}" ] || die "version missing in ${meta}"

stage=$(mktemp -d)
trap 'rm -rf "${stage}"' EXIT
cp -a "${src}/rootfs/." "${stage}/"
mkdir -p "${stage}/usr/lib/extension-release.d"
{
  echo "ID=${id:-flatcar}"
  [ -z "${level}" ] || echo "SYSEXT_LEVEL=${level}"
  echo "ARCHITECTURE=${arch:-x86-64}"
  echo "VERSION_ID=${version}"
} > "${stage}/usr/lib/extension-release.d/extension-release.${name}"

mkdir -p "${outdir}"
out="${outdir}/${name}-${version}-${arch:-x86-64}.raw"
mksquashfs "${stage}" "${out}" -quiet -no-progress -noappend -all-root \
  -mkfs-time 0 -all-time 0 -no-xattrs >&2
echo "sha512: $({ sha512sum "${out}" 2>/dev/null || shasum -a 512 "${out}"; } | cut -d' ' -f1)" >&2
echo "${out}"
