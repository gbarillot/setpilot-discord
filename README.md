# SetPilot Discord agent

## Deploy
git clone <repo>
cd <repo>
cp .env.example .env
nano .env
docker compose up -d --build

Check logs:
docker compose logs -f bot