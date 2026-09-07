# Deploy: fly.io sprite

Inventory runs on a fly.io [sprite](https://sprites.dev) — a lightweight, single-machine dev environment on Ubuntu with a public HTTP URL. Postgres and the API both run inside the sprite, managed as sprite services.

## Live URL

`https://inventory-b2mxg.sprites.app` — auth `public` (bearer api key gates access).

- `GET /health` — unauthenticated
- Everything else — `Authorization: Bearer <inv_...>`
- MCP: `https://inventory-b2mxg.sprites.app/mcp`

## First deploy (recipe)

```sh
# 1. Create the sprite (creates isolated Ubuntu env, gets a URL)
sprite create --skip-console inventory
sprite use inventory

# 2. Install postgres + dbmate
sprite exec -- bash -lc 'sudo apt-get update -qq && sudo apt-get install -y -qq postgresql postgresql-contrib git'
sprite exec -- bash -lc 'sudo curl -fsSL -o /usr/local/bin/dbmate \
  https://github.com/amacneil/dbmate/releases/latest/download/dbmate-linux-amd64 \
  && sudo chmod +x /usr/local/bin/dbmate'

# 3. Wrap postgres in a foreground launcher and register as a sprite service
sprite exec -- bash -lc 'sudo tee /usr/local/bin/run-postgres.sh <<EOF
#!/usr/bin/env bash
set -euo pipefail
exec sudo -u postgres /usr/lib/postgresql/18/bin/postgres \
  -D /var/lib/postgresql/18/main \
  -c config_file=/etc/postgresql/18/main/postgresql.conf
EOF
sudo chmod +x /usr/local/bin/run-postgres.sh'

sprite exec -- bash -lc '/.sprite/bin/sprite-env services create postgres \
  --cmd /usr/local/bin/run-postgres.sh --no-stream'

# 4. Create db + role
sprite exec -- bash -lc 'sudo -u postgres createuser -s sprite; \
  sudo -u postgres createdb -O sprite inventory'

# 5. Clone the repo, set DATABASE_URL, migrate, build
sprite exec -- bash -lc 'cd ~ && git clone https://github.com/KevinMcHugh/inventory.git'
sprite exec -- bash -lc "echo 'export DATABASE_URL=\"postgres:///inventory?host=/var/run/postgresql&sslmode=disable\"' >> ~/.profile"
sprite exec -- bash -lc 'cd ~/inventory && dbmate up && go build -o bin/server ./cmd/server'

# 6. Bootstrap first tenant + api key (SAVE THE PRINTED KEY)
sprite exec -- bash -lc 'cd ~/inventory && ./bin/server bootstrap --tenant "<your name>"'

# 7. Register the API as a sprite service on the proxy port (8080)
sprite exec -- bash -lc '/.sprite/bin/sprite-env services create inventory \
  --cmd /home/sprite/inventory/bin/server \
  --dir /home/sprite/inventory \
  --needs postgres --http-port 8080 \
  --env "DATABASE_URL=postgres:///inventory?host=/var/run/postgresql&sslmode=disable,PORT=8080" \
  --no-stream'

# 8. Make the URL publicly reachable so Claude.ai / external clients can hit MCP
sprite config update -s inventory --url-auth public
```

## Notes on the sprite runtime

- **Sprites pause when idle** and wake on incoming HTTP requests (the proxy auto-starts the `--http-port` service). Services with `--needs postgres` bring postgres up first.
- **State is a writable overlay** on the base Ubuntu image. It survives pauses and restarts, but not destruction. Take a checkpoint (`sprite-env checkpoints create`) before risky changes.
- **Env vars for services** live on the service definition (`--env`); shell profile env vars only apply to interactive `sprite exec` sessions.
- **`--http-port` may only be set on one service.** That is the inventory server here; postgres listens only on localhost + unix socket.

## Redeploy after a code change

```sh
# On your machine
git push

# In the sprite
sprite exec -- bash -lc 'cd ~/inventory && git pull && \
  dbmate up && \
  go build -o bin/server ./cmd/server && \
  /.sprite/bin/sprite-env services restart inventory'
```

## Rotate the bootstrap key

The first key printed by `bootstrap` is stored as a SHA-256 hash — plaintext is unrecoverable. Rotate with:

```sh
sprite exec -- bash -lc 'cd ~/inventory && ./bin/server keys list --tenant <tenant-id>'
sprite exec -- bash -lc 'cd ~/inventory && ./bin/server keys rotate --key-id <key-id>'
```

## Connecting Claude

Point Claude Desktop or `claude.ai/code` MCP settings at:

- **URL:** `https://inventory-b2mxg.sprites.app/mcp`
- **Auth:** paste the raw `inv_...` key as the bearer token

## Rolling back

```sh
sprite-env checkpoints list
sprite restore <checkpoint-id>
```

## Tearing down

```sh
sprite destroy inventory
```
