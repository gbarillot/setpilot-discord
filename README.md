# SetPilot Discord agent

The agent is a Go service backed by a read-only SQLite database.

## Test a question without Discord

Run a single question from this directory against a current SQLite database:

```sh
go run . -question "Quels sont les présents à la prochaine répet?" -db /path/to/setpilot.db -debug
```

The answer goes to stdout; `-debug` writes the processing path and participant
count (or generated SQL and row count for other questions) to stderr. Use
`-now 2026-10-19T12:00:00Z` to reproduce a question at a specific time. If
`-db` is omitted, the path comes from `SQLITE_PATH` in `.env`. The dedicated
next-rehearsal participants question needs only the database; other questions
also need an OpenRouter key in `.env`. This mode never connects to Discord.

The Docker setups mount Fiber's local `../fiber/setpilot.db` at
`/home/bot/fiber/setpilot.db` in read-only mode. For host-side CLI runs, pass
`-db ../fiber/setpilot.db` (the Docker path in `.env` is not a host path).

## Whitelist configuration

`WHITE_LIST_WORDS` in the bot's `.env` is the Discord mention filter. The bot
reads it at startup, so changes require a restart. There is no second
hard-coded whitelist. `.env.example` is a template, not a runtime source.

- Local runs: this directory's `.env` (unless `SET_PILOT_ENV_FILE` is set).
- Production systemd service: `/home/setpilot/agent/.env` (the service runs
  from that directory). Restart with `sudo systemctl restart setpilot-agent`.
- Docker Compose: `./.env`, mounted inside the container as `/home/bot/.env`.

Keep the full list of words in each deployment's `.env`; updating only
`.env.example` does not change a running bot. The bare `repet` and `répet`
entries also support older deployed binaries that split trailing apostrophes.

## Docker Compose deployment

Copy `.env.example` to `.env` and configure the secrets and SQLite path. The
Compose file mounts `../fiber` at `/home/bot/fiber` and `.env` at `/home/bot/.env`.
Then run `docker compose up -d --build`.

Check logs:
docker compose logs -f bot
