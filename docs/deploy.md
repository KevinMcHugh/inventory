# Deploy: fly.io sprite

Inventory runs on a fly.io [sprite](https://sprites.dev) — a lightweight, single-machine dev environment on Ubuntu with a public HTTP URL. Postgres and the API both run inside the sprite, managed as sprite services.

## Live URL

`https://inventory-b2mxg.sprites.app` — auth `public` (bearer api key gates access).

- `GET /health` — unauthenticated
- Everything else — `Authorization: Bearer <inv_...>`
- MCP: `https://inventory-b2mxg.sprites.app/mcp/rpc`

## First deploy (recipe)

```sh
# 1. Create the sprite (isolated Ubuntu env, gets a URL)
sprite create --skip-console inventory
sprite use inventory

# 2. Install postgres, git, and dbmate
sprite exec -- bash -lc 'sudo apt-get update -qq && sudo apt-get install -y -qq postgresql postgresql-contrib git'
sprite exec -- bash -lc 'sudo curl -fsSL -o /usr/local/bin/dbmate \
  https://github.com/amacneil/dbmate/releases/latest/download/dbmate-linux-amd64 \
  && sudo chmod +x /usr/local/bin/dbmate'

# 3. Clone the repo (needed for the postgres launcher script in step 4 and
#    the app source in step 7)
sprite exec -- bash -lc 'cd ~ && git clone https://github.com/KevinMcHugh/inventory.git'

# 4. Install the postgres launcher and register it as a sprite service.
#    The launcher (deploy/run-postgres.sh in this repo) recreates
#    /var/run/postgresql before exec — that path is tmpfs on the sprite base
#    image and gets wiped on every reboot, so postgres cannot start without it.
sprite exec -- bash -lc 'sudo install -m 0755 ~/inventory/deploy/run-postgres.sh /usr/local/bin/run-postgres.sh'
sprite exec -- bash -lc '/.sprite/bin/sprite-env services create postgres \
  --cmd /usr/local/bin/run-postgres.sh --no-stream'

# 5. Create db + role
sprite exec -- bash -lc 'sudo -u postgres createuser -s sprite; \
  sudo -u postgres createdb -O sprite inventory'

# 6. Migrate, build the server, and (optionally) build the web UI
sprite exec -- bash -lc "echo 'export DATABASE_URL=\"postgres:///inventory?host=/var/run/postgresql&sslmode=disable\"' >> ~/.profile"
sprite exec -- bash -lc 'cd ~/inventory && dbmate up && go build -o bin/server ./cmd/server'
sprite exec -- bash -lc 'cd ~/inventory/web && npm install --silent && npm run build'

# 7. Bootstrap first tenant + api key (SAVE THE PRINTED KEY)
sprite exec -- bash -lc 'cd ~/inventory && ./bin/server bootstrap --tenant "<your name>"'

# 8. Register the API as a sprite service on the proxy port (8080).
#    PUBLIC_URL is the origin OAuth discovery documents advertise, so it
#    must match the reachable https URL.
sprite exec -- bash -lc '/.sprite/bin/sprite-env services create inventory \
  --cmd /home/sprite/inventory/bin/server \
  --dir /home/sprite/inventory \
  --needs postgres --http-port 8080 \
  --env "DATABASE_URL=postgres:///inventory?host=/var/run/postgresql&sslmode=disable,PORT=8080,PUBLIC_URL=https://<sprite-name>-<org>.sprites.app" \
  --no-stream'

# 9. Make the URL publicly reachable so Claude.ai / external clients can hit MCP
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
  (cd web && npm run build) && \
  /.sprite/bin/sprite-env services restart inventory'
```

If `deploy/run-postgres.sh` changes, re-install it and restart postgres:

```sh
sprite exec -- bash -lc 'sudo install -m 0755 ~/inventory/deploy/run-postgres.sh /usr/local/bin/run-postgres.sh && \
  /.sprite/bin/sprite-env services restart postgres'
```

## Rotate the bootstrap key

The first key printed by `bootstrap` is stored as a SHA-256 hash — plaintext is unrecoverable. Rotate with:

```sh
sprite exec -- bash -lc 'cd ~/inventory && ./bin/server keys list --tenant <tenant-id>'
sprite exec -- bash -lc 'cd ~/inventory && ./bin/server keys rotate --key-id <key-id>'
```

## Connecting Claude

Point Claude Desktop or `claude.ai/code` MCP settings at:

- **URL:** `https://inventory-b2mxg.sprites.app/mcp/rpc`
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
