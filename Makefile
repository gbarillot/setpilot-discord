RELEASE_TS := $(shell date +%Y%m%d%H%M%S)
SERVER ?= setpilot@188.245.187.59
REMOTE_AGENT_DIR ?= /home/setpilot/agent
SERVICE_NAME ?= setpilot-agent
AGENT_CONTAINER ?= agent-api-1
AGENT_BINARY ?= setpilot-agent-linux-amd64

.PHONY: build
build:
	docker exec -it $(AGENT_CONTAINER) sh -c "cd /home/bot && GOOS=linux GOARCH=amd64 go build -o $(AGENT_BINARY)"

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

.PHONY: clear
clear:
	docker system prune -af

.PHONY: test
test:
	docker exec -it $(AGENT_CONTAINER) sh -c "cd /home/bot && go test ./..."

.PHONY: shell
shell:
	docker exec -it $(AGENT_CONTAINER) bash

.PHONY: deploy
deploy: build
	scp $(AGENT_BINARY) $(SERVER):$(REMOTE_AGENT_DIR)/releases/$(RELEASE_TS)
	ssh $(SERVER) "ln -sfn $(REMOTE_AGENT_DIR)/releases/$(RELEASE_TS) $(REMOTE_AGENT_DIR)/agent"
	ssh -t $(SERVER) "sudo systemctl restart $(SERVICE_NAME)"
