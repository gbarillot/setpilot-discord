.PHONY: build
DEPLOY_HOST ?= setpilot@188.245.187.59
DEPLOY_DIR ?= /home/setpilot/agent
DEPLOY_BRANCH ?= main

build:
	docker compose -f .devcontainer/compose.yaml -p agent build

.PHONY: start
start:
	docker compose -f .devcontainer/compose.yaml -p agent up -d

.PHONY: stop
stop:
	docker compose -f .devcontainer/compose.yaml -p agent down

.PHONY: restart
restart:
	docker compose -f .devcontainer/compose.yaml -p agent down
	docker compose -f .devcontainer/compose.yaml -p agent up -d

.PHONY: shell
shell:
	docker exec -it agent-api-1 bash

.PHONY: logs
logs:
	docker compose -f .devcontainer/compose.yaml -p agent logs -f

.PHONY: clear
clear:
	docker system prune -af

.PHONY: deploy
deploy:
	ssh $(DEPLOY_HOST) "cd $(DEPLOY_DIR) && git fetch origin $(DEPLOY_BRANCH) && git checkout $(DEPLOY_BRANCH) && git pull --ff-only origin $(DEPLOY_BRANCH) && mkdir -p logs && if ! grep -q '^HOST_UID=' .env; then printf '\nHOST_UID=%s\n' \$$(id -u) >> .env; fi && if ! grep -q '^HOST_GID=' .env; then printf 'HOST_GID=%s\n' \$$(id -g) >> .env; fi && sudo chown -R \$$(id -u):\$$(id -g) logs && docker-compose down && docker-compose up -d --build && docker-compose logs --tail=100 bot"
