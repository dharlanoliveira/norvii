.PHONY: bootstrap local-start local-status local-stop persistence persistence-config \
	persistence-health persistence-integration persistence-migrate persistence-migration-status \
	persistence-initialize-snapshots \
	persistence-reset persistence-stop persistence-up persistence-verify \
	persistence-verify-api persistence-verify-ingestion

PERSISTENCE_ENV_FILE := infra/.env
PERSISTENCE_COMPOSE := docker compose --env-file $(PERSISTENCE_ENV_FILE) -f infra/compose.yaml
PERSISTENCE_RUN := python infra/scripts/run-with-environment.py $(PERSISTENCE_ENV_FILE)
LOCAL_ENVIRONMENT_MANAGER := python infra/scripts/manage-local-environment.py

bootstrap: local-start

local-start:
	@$(LOCAL_ENVIRONMENT_MANAGER) start

local-status:
	@$(LOCAL_ENVIRONMENT_MANAGER) status

local-stop:
	@$(LOCAL_ENVIRONMENT_MANAGER) stop

persistence:
	@$(MAKE) persistence-up
	@$(MAKE) persistence-migrate
	@$(MAKE) persistence-verify

persistence-config:
	@test -f $(PERSISTENCE_ENV_FILE) || { echo "Missing $(PERSISTENCE_ENV_FILE); copy infra/.env.example and replace secret markers." >&2; exit 1; }
	@$(PERSISTENCE_COMPOSE) config --services

persistence-up:
	@$(MAKE) persistence-config
	@$(PERSISTENCE_COMPOSE) up --detach --wait --wait-timeout 120
	@$(MAKE) persistence-health

persistence-health:
	@bash infra/scripts/inspect-health.sh $(PERSISTENCE_ENV_FILE)

persistence-migrate:
	@$(PERSISTENCE_RUN) $(MAKE) -C apps/api migrate

persistence-initialize-snapshots:
	@$(PERSISTENCE_RUN) $(MAKE) -C apps/api initialize-snapshots

persistence-migration-status:
	@$(PERSISTENCE_RUN) $(MAKE) -C apps/api migration-status

persistence-verify:
	@$(MAKE) persistence-verify-api
	@$(MAKE) persistence-verify-ingestion

persistence-verify-api:
	@$(PERSISTENCE_RUN) $(MAKE) -C apps/api verify-persistence

persistence-verify-ingestion:
	@$(PERSISTENCE_RUN) $(MAKE) -C apps/ingestion verify-persistence

persistence-stop:
	@$(PERSISTENCE_COMPOSE) down --remove-orphans

persistence-reset:
	@bash infra/scripts/reset-local-data.sh $(PERSISTENCE_ENV_FILE) "$(CONFIRM)"

persistence-integration:
	@bash infra/scripts/verify-foundation.sh
