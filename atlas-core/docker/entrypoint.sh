#!/bin/sh
# Atlas Core container entrypoint.
#
# Docker mounts named volumes (atlas-core-object-storage) as root:root by
# default, which would prevent the unprivileged atlas user from writing into
# /data/object-storage. We fix ownership on every container start (idempotent
# and cheap when already correct) and then drop to UID 10001 via su-exec
# before exec'ing the actual Atlas Core binary. From that point on, no
# privileged process exists.
#
# If the container is already running as the atlas user (e.g. a developer
# overrode `--user`), we skip the chown step and just exec.

set -eu

ATLAS_HOME=/app
ATLAS_DATA_ROOT="${ATLAS_CORE_OBJECT_STORAGE_ROOT:-/data/object-storage}"
ATLAS_USER=atlas
ATLAS_UID=10001

if [ "$(id -u)" = "0" ]; then
    mkdir -p "$ATLAS_DATA_ROOT"
    chown -R "${ATLAS_UID}:${ATLAS_UID}" "$ATLAS_DATA_ROOT"
    exec su-exec "${ATLAS_UID}:${ATLAS_UID}" "$@"
fi

exec "$@"
