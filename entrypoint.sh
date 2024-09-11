#! /bin/sh

set -eu

PUID=${PUID:-1000}
PGID=${PGID:-1000}

if ! getent group "$PGID" >/dev/null; then
  addgroup -g "$PGID" pixiv
fi

if ! getent passwd "$PUID" >/dev/null; then
  adduser -u "$PUID" -G pixiv --disabled-password --gecos "" pixiv
fi

chown -R "$PUID":"$PGID" /pixiv-dl

exec su-exec "$PUID":"$PGID" "$@"
