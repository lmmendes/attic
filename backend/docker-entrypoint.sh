#!/bin/sh
set -eu

storage_path=${ATTIC_LOCAL_STORAGE_PATH:-/data/uploads}
current_uid=$(id -u)
current_gid=$(id -g)

# Honor the runtime identity unless root needs to drop privileges.
run_uid=$current_uid
run_gid=$current_gid
if [ "$current_uid" -eq 0 ]; then
	run_uid=$(id -u appuser)
	run_gid=$(id -g appuser)
fi

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
	if [ "$current_uid" -eq 0 ]; then
		chown "$run_uid:$run_gid" "$storage_path"
	elif [ "$run_uid" -ne "$current_uid" ] || [ "$run_gid" -ne "$current_gid" ]; then
		echo "ATTIC_PUID/ATTIC_PGID must match the container UID/GID when running as non-root (current: ${current_uid}:${current_gid})" >&2
		exit 1
	fi
fi

case "${1:-}" in
	-*) set -- /app/attic "$@" ;;
esac

if [ "$current_uid" -eq 0 ]; then
	exec su-exec "$run_uid:$run_gid" "$@"
fi

exec "$@"
