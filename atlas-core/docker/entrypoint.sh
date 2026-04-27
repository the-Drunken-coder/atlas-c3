#!/bin/sh
# Atlas Core container entrypoint.
#
# Docker mounts named volumes (atlas-core-object-storage) as root:root by
# default, which would prevent the unprivileged atlas user from writing into
# /data/object-storage. On first start we fix ownership of the volume root and
# then drop to UID 10001 via su-exec before exec'ing the actual Atlas Core
# binary. From that point on, no privileged process exists.
#
# We only chown -R when the top-level directory ownership is wrong. Recursive
# chown over the entire object storage on every container start would be
# O(number of files) and grows expensive as the store fills; gating it on the
# top-level owner makes steady-state startup O(1).
#
# If the container is already running as the atlas user (e.g. a developer
# overrode `--user`), we skip the chown step and just exec.

set -eu

ATLAS_DATA_ROOT="${ATLAS_CORE_OBJECT_STORAGE_ROOT:-/data/object-storage}"
ATLAS_UID=10001
ATLAS_OWNER="${ATLAS_UID}:${ATLAS_UID}"

if [ "$(id -u)" = "0" ]; then
    mkdir -p "$ATLAS_DATA_ROOT"
    current_owner=$(stat -c '%u:%g' "$ATLAS_DATA_ROOT")
    if [ "$current_owner" != "$ATLAS_OWNER" ]; then
        chown -R "$ATLAS_OWNER" "$ATLAS_DATA_ROOT"
    fi
    exec su-exec "$ATLAS_OWNER" "$@"
fi

exec "$@"
