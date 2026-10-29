#!/usr/bin/env bash
# Rebuild and restart inventory in place on the fly.io sprite.
#
# Run from the sprite after the checkout is already at the commit to deploy
# (deploy/sprite-deploy does the fetch, then execs this). Expects a login
# shell so DATABASE_URL, go, npm, and dbmate come from ~/.profile.
#
# The binary is built to a temp path and swapped in only after the build
# succeeds, so a broken build leaves the running server untouched.
set -euo pipefail

cd "$(dirname "$0")/.."

echo ">> deploying $(git rev-parse --short HEAD)"

echo ">> migrating"
dbmate up

echo ">> building server"
go build -o bin/server.new ./cmd/server

echo ">> building web"
(cd web && npm install --silent --no-audit --no-fund && npm run build)

if ! cmp -s deploy/run-postgres.sh /usr/local/bin/run-postgres.sh; then
  echo ">> run-postgres.sh changed; reinstalling and restarting postgres"
  sudo install -m 0755 deploy/run-postgres.sh /usr/local/bin/run-postgres.sh
  /.sprite/bin/sprite-env services restart postgres
fi

mv -f bin/server.new bin/server

echo ">> restarting inventory"
/.sprite/bin/sprite-env services restart inventory

for _ in $(seq 1 30); do
  if curl -fsS http://localhost:8080/health >/dev/null 2>&1; then
    echo ">> healthy"
    exit 0
  fi
  sleep 1
done

echo "!! inventory did not become healthy within 30s" >&2
exit 1
