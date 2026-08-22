# SetPilot Discord agent

The agent is a Go service backed by a read-only SQLite database.

## Deploy
git clone <repo>
cd <repo>
cp .env.example .env
Put the migrated SQLite database at `data/setpilot.sqlite3`.
nano .env
docker compose up -d --build

Check logs:
docker compose logs -f bot
