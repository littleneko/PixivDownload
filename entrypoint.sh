#! /bin/sh

set -eu

PUID=${PUID:-1000}
PGID=${PGID:-1000}

gname=pixiv
if ! getent group "$PGID" >/dev/null; then
  addgroup -g "$PGID" $gname
else
  gname=$(getent group "$PGID" | cut -d: -f1)
fi

if ! getent passwd "$PUID" >/dev/null; then
  adduser -u "$PUID" -G $gname --disabled-password --gecos "" pixiv
fi

chown -R "$PUID":"$PGID" /pixiv-dl

exec su-exec "$PUID":"$PGID" "$@"
