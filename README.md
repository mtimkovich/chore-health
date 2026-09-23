# Chore Health

Tracks chores as countdowns, styled after the Roomba "Product Health" screen — each
chore is a card with a draining progress bar and an "~N HRS LEFT" readout, sorted so
the most urgent chore is always on top.

- **Backend**: Go (`net/http`, SQLite via `modernc.org/sqlite`, no cgo needed)
- **Frontend**: Vue 3 + Vite
- **DB**: SQLite file at `backend/chores.db` (created automatically on first run)

## Run it

One-time setup:

```bash
npm install
npm install --prefix frontend
```

Then, from the repo root:

```bash
npm run dev
```

This runs the Go backend and the Vite dev server together (via `concurrently`),
with both auto-reloading on save:

- **Backend**: `nodemon` watches `backend/**/*.go` and restarts `go run .` on change.
- **Frontend**: Vite's own dev server hot-reloads `.vue`/`.js`/`.css` changes.

The frontend is on `http://localhost:5173` and proxies `/api` requests to the Go
backend on `http://localhost:8080` — open the frontend URL in a browser.

To run them separately instead: `npm run dev:backend` / `npm run dev:frontend`,
or `cd backend && go run .` / `cd frontend && npm run dev`.

### Docker

```bash
docker build -t chore-health .
docker run -d -p 8080:8080 -v chore-health-data:/data --name chore-health chore-health
```

Or with `docker-compose.yml` (bind-mounts `./data` instead of a named
volume, and builds the image itself):

```bash
docker compose up -d --build
```

Open `http://localhost:8080` — one container serves both the API and the
built frontend (the Go binary embeds it at build time). `chores.db` lives at
`/data/chores.db`, backed by whichever of the above you used, so it survives
container restarts/recreates.

The container runs as a non-root user (fixed UID/GID `10001`, set in the
Dockerfile). A **named volume** (`-v chore-health-data:/data`) is initialized
from the image, so this just works. A **host bind mount** (`-v ./data:/data`,
including the compose file above) is not - Docker mounts the host path as-is,
ownership and all, so `./data` needs to actually be owned by `10001` or the
container can't write to it (SQLite fails with "attempt to write a readonly
database"). Fix it once, before first starting the container:

```bash
mkdir -p data
sudo chown -R 10001:10001 data
```

Useful env vars (see `docker run -e NAME=value ...`):

- `CHORE_HEALTH_PASSWORD` — see [Password protection](#password-protection) below.
- `TZ` (e.g. `TZ=America/New_York`) — containers default to UTC, and this app's
  "back at local midnight" logic (see [Model](#model)) needs the real
  timezone to mean anything.

## Model

Each chore has a name, an `interval_hours` (how long it's allowed to go between
completions), and a `recurring` flag:

- **Recurring** chores reset their countdown when marked done, and move to the
  Completed tab until local midnight.
- **Non-recurring** chores move to the Completed tab when marked done and are
  permanently deleted a week later.

Any chore can also be deleted outright regardless of its recurring setting.

`percent_remaining` and `hours_left` are computed server-side from
`last_completed_at` + `interval_hours` on every request, so the UI just needs to
poll `GET /api/chores` (it does this once a minute) to stay current.

## Password protection

Optional, and off by default — with no password set, the app works exactly as
if this feature didn't exist. There's no in-app way to set one (by design -
it's not exposed to the frontend at all): set the `CHORE_HEALTH_PASSWORD` env
var before starting the backend. It's read once at startup, never persisted
to the database, and unsetting it (or leaving it unset) disables auth
entirely.

Once a password is set, the app shows a login screen and a logout button
appears in the header. Sessions are stored in the database and last 90 days
(matching the cookie), so a server restart doesn't log anyone out.
