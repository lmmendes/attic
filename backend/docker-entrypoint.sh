#!/bin/sh
set -eu

storage_path=${ATTIC_LOCAL_STORAGE_PATH:-/data/uploads}
run_uid=$(id -u appuser)
run_gid=$(id -g appuser)

if [ -z "${ATTIC_S3_ACCESS_KEY:-}" ] || [ -z "${ATTIC_S3_SECRET_KEY:-}" ]; then
	if [ -n "${ATTIC_PUID:-}" ] || [ -n "${ATTIC_PGID:-}" ]; then
		if [ -z "${ATTIC_PUID:-}" ] || [ -z "${ATTIC_PGID:-}" ]; then
			echo "ATTIC_PUID and ATTIC_PGID must be set together" >&2
			exit 1
		fi

		case "${ATTIC_PUID}:${ATTIC_PGID}" in
			*[!0-9:]* | *:*:*)
				echo "ATTIC_PUID and ATTIC_PGID must be non-negative integers" >&2
				exit 1
				;;
		esac

		run_uid=${ATTIC_PUID}
		run_gid=${ATTIC_PGID}
	fi

	mkdir -p "$storage_path"
	chown "$run_uid:$run_gid" "$storage_path"
fi

case "${1:-}" in
	-*) set -- /app/attic "$@" ;;
esac

exec su-exec "$run_uid:$run_gid" "$@"
