.PHONY: build
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
