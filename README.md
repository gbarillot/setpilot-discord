# SetPilot Discord agent

The agent is a Go service backed by a read-only SQLite database.

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
Compose file mounts `../data` at `/home/bot/data` and `.env` at `/home/bot/.env`.
Then run `docker compose up -d --build`.

Check logs:
docker compose logs -f bot
