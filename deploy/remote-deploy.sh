#!/bin/sh
set -eu

LC_ALL=C
export LC_ALL

REGISTRY=ghcr.io
REGISTRY_USER=deniskorkmaz
DEPLOY_DIR=/opt/nostra
LOCK_FILE=/var/lock/nostra-deploy
PRUNE_AGE=336h

sha="${SSH_ORIGINAL_COMMAND:-}"

case "$sha" in
*[!0123456789abcdef]* | "")
	echo "nostra-deploy: expected a 40-character commit sha" >&2
	exit 1
	;;
esac

if [ ${#sha} -ne 40 ]; then
	echo "nostra-deploy: expected a 40-character commit sha" >&2
	exit 1
fi

exec 9>"$LOCK_FILE"
if ! flock --nonblock 9; then
	echo "nostra-deploy: another deployment is in progress" >&2
	exit 1
fi

cd "$DEPLOY_DIR"

if ! grep -q '^NOSTRA_VERSION=' .env; then
	echo "nostra-deploy: .env has no NOSTRA_VERSION line" >&2
	exit 1
fi

docker login "$REGISTRY" --username "$REGISTRY_USER" --password-stdin

NOSTRA_VERSION="$sha" docker compose pull server migrate caddy

sed -i "s|^NOSTRA_VERSION=.*|NOSTRA_VERSION=$sha|" .env

docker compose up -d --no-build --remove-orphans

docker logout "$REGISTRY"

docker image prune -af --filter "until=$PRUNE_AGE" >/dev/null

logger -t nostra-deploy "deployed $sha"

printf 'compose-sha256 %s\n' "$(sha256sum <compose.yaml | cut -d' ' -f1)"
