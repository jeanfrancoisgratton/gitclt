#!/usr/bin/env sh
# version-bump.sh -- synchronize gitclt's version across
# gitclt.json and every present __<distro> packaging file,
# resetting each one's own release/pkgrel counter back to 1.
#
# Deliberately does NOT touch src/cmd/root.go: the binary reads its version
# via -ldflags -X at build time (see the buildVersion/buildDate vars there),
# so a version bump genuinely only requires running this one script.
#
# Never touches __redhat/gitclt.spec's %changelog -- that is
# owned by __redhat/updateChangelog.sh / `make commitcl`, not by this script.

set -eu

SOFTWARE_NAME="gitclt"
MANIFEST="${SOFTWARE_NAME}.json"
DEFAULT_RELEASE=1

usage() {
	cat <<-EOF >&2
	Usage: $0 NEW_VERSION

	  Bumps ${MANIFEST}'s versionnumber and every present __<distro>
	  packaging file's version field, resetting each one's own
	  release/pkgrel counter to ${DEFAULT_RELEASE}.
	EOF
	exit "${1:-1}"
}

case "${1-}" in
	-h|--help) usage 0 ;;
	-*)
		echo "$0: '$1' looks like an option, not a version" >&2
		usage 1
		;;
esac
[ "$#" -eq 1 ] || usage 1

NEWVER="$1"

case "$NEWVER" in
	[0-9]*) : ;;
	*)
		echo "$0: version must start with a digit: '$NEWVER'" >&2
		exit 1
		;;
esac

case "$NEWVER" in
	*[!A-Za-z0-9._~-]*)
		echo "$0: version contains invalid characters: '$NEWVER'" >&2
		exit 1
		;;
esac

if [ ! -f "$MANIFEST" ]; then
	echo "$0: $MANIFEST not found; run this from the project root" >&2
	exit 1
fi

SPEC="__redhat/${SOFTWARE_NAME}.spec"

FILES="$MANIFEST"
[ -f __alpine/APKBUILD ]    && FILES="$FILES __alpine/APKBUILD"
[ -f __archlinux/PKGBUILD ] && FILES="$FILES __archlinux/PKGBUILD"
[ -f __debian/control ]     && FILES="$FILES __debian/control"
[ -f "$SPEC" ]              && FILES="$FILES $SPEC"

if git rev-parse --git-dir >/dev/null 2>&1; then
	# shellcheck disable=SC2086
	dirty=$(git status --porcelain -- $FILES)
	if [ -n "$dirty" ]; then
		echo "$0: refusing to run -- uncommitted changes in a file this script touches:" >&2
		echo "$dirty" >&2
		exit 1
	fi
fi

# patch_line PATTERN REPLACEMENT FILE : replace the (unique) line starting
# with PATTERN with REPLACEMENT, preserving FILE's mode across the
# temp-file+mv swap (mv onto an existing path replaces the inode, which
# silently drops the executable bit otherwise), then verify the patch landed.
patch_line() {
	pattern="$1"; replacement="$2"; file="$3"
	mode=$(stat -c '%a' "$file" 2>/dev/null || stat -f '%Lp' "$file")
	tmp=$(mktemp)
	sed "s|^${pattern}.*|${replacement}|" "$file" >"$tmp"
	mv "$tmp" "$file"
	chmod "$mode" "$file"
	grep -qF "$replacement" "$file" || {
		echo "$0: failed to patch $file (expected to find: $replacement)" >&2
		exit 1
	}
}

patch_line '  "versionnumber":' "  \"versionnumber\": \"${NEWVER}\"," "$MANIFEST"
patch_line '  "defaultreleasenumber":' "  \"defaultreleasenumber\": \"${DEFAULT_RELEASE}\"," "$MANIFEST"

if [ -f __alpine/APKBUILD ]; then
	patch_line 'pkgver=' "pkgver=${NEWVER}" __alpine/APKBUILD
	patch_line 'pkgrel=' "pkgrel=${DEFAULT_RELEASE}" __alpine/APKBUILD
fi

if [ -f __archlinux/PKGBUILD ]; then
	patch_line 'pkgver=' "pkgver=${NEWVER}" __archlinux/PKGBUILD
	patch_line 'pkgrel=' "pkgrel=${DEFAULT_RELEASE}" __archlinux/PKGBUILD
fi

if [ -f __debian/control ]; then
	patch_line 'Version:' "Version: ${NEWVER}-${DEFAULT_RELEASE}" __debian/control
fi

if [ -f "$SPEC" ]; then
	patch_line '%define _version' "%define _version ${NEWVER}" "$SPEC"
	patch_line '%define _rel'     "%define _rel ${DEFAULT_RELEASE}" "$SPEC"
fi

echo "Bumped ${SOFTWARE_NAME} to ${NEWVER} (release reset to ${DEFAULT_RELEASE} everywhere it's tracked)."
