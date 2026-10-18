#!/usr/bin/env bash
# dontexec.sh -- create or remove the _dontexec marker file
#
# Lives at the root of the repo; every path it touches is relative to that
# root, so the script can be dropped into any repo unchanged.
#
#   dontexec.sh             create _dontexec in every __* subdirectory
#   dontexec.sh <dir>       create _dontexec in <dir>
#   dontexec.sh -r          remove _dontexec from every __* subdirectory
#   dontexec.sh -r <dir>    remove _dontexec from <dir>

set -euo pipefail

# the repo root is wherever this script sits; work from there so that
# everything below is a plain relative path
cd "$(dirname "$0")"

MARKER="_dontexec"

usage() {
	cat >&2 <<-EOF
	Usage: ${0##*/} [-r] [directory]

	  -r  remove the ${MARKER} file instead of creating it

	With no <directory>, the operation applies to every __* subdirectory of
	the repo root. <directory>, if given, must be one of:
	$(printf '  %s\n' __*/)
	EOF
	exit 1
}

remove=0

while getopts ":r" opt; do
	case "${opt}" in
		r) remove=1 ;;
		*) usage ;;
	esac
done
shift $((OPTIND - 1))

(($# <= 1)) || usage

if (($# == 1)); then
	target="${1#./}"    # tolerate ./__debian
	target="${target%/}" # tolerate __debian/
	# only the repo root's own __* subdirectories are legal targets: no
	# absolute paths, no traversal, no nesting
	if [[ "${target}" != __* || "${target}" == */* ]]; then
		echo "${0##*/}: ${1}: not a __* subdirectory of the repo root" >&2
		exit 1
	fi
	targets=("${target}")
else
	targets=(__*/)
	targets=("${targets[@]%/}")
fi

for target in "${targets[@]}"; do
	if [[ ! -d "${target}" ]]; then
		echo "${0##*/}: ${target}: no such directory" >&2
		exit 1
	fi

	label="${target}/${MARKER}"

	if ((remove)); then
		rm -f -- "${label}"
		echo "removed ${label}"
	else
		touch -- "${label}"
		echo "created ${label}"
	fi
done
