# Chore Timer

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

## Model

Each chore has a name, an `interval_hours` (how long it's allowed to go between
completions), and a `recurring` flag:

- **Recurring** chores reset their countdown when marked done, and move to the
  Completed tab until local midnight.
- **Non-recurring** chores move to the Completed tab when marked done and are
  permanently deleted 24 hours later.

Any chore can also be deleted outright regardless of its recurring setting.

`percent_remaining` and `hours_left` are computed server-side from
`last_completed_at` + `interval_hours` on every request, so the UI just needs to
poll `GET /api/chores` (it does this once a minute) to stay current.

## Password protection

Optional, and off by default — with no password set, the app works exactly as
if this feature didn't exist. There's no in-app way to set one (by design -
it's not exposed to the frontend at all); instead:

- **Env var**: set `CHORE_TIMER_PASSWORD` before starting the backend. It's
  applied to the database on every startup, so it always wins over whatever
  is in the database.
- **Directly in the database**: with the backend stopped (or at least not
  mid-write), set it with sqlite3:

  ```bash
  sqlite3 backend/chores.db "INSERT INTO settings (key, value) VALUES ('password', 'yourpassword') ON CONFLICT(key) DO UPDATE SET value = excluded.value;"
  ```

  Clear it the same way with `DELETE FROM settings WHERE key = 'password';`.

Either way it's stored in plaintext in `chores.db` - this is a local-network
tool, not a public one, so that's an intentional simplification, not an
oversight. Once a password is set, the app shows a login screen and a logout
button appears in the header; sessions are an in-memory cookie, so restarting
the backend logs everyone out.
