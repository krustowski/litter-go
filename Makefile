#
# litter-go / Makefile
#

#
# VARS
#

include .env.example
-include .env

PROJECT_NAME?=litter-go

DOCKER_IMAGE_TAG?=${PROJECT_NAME}-image
DOCKER_CONTAINER_NAME?=${PROJECT_NAME}-server

GOARCH := $(shell go env GOARCH)
GOCACHE?=/home/${USER}/.cache/go-build
GOMODCACHE?=/home/${USER}/go/pkg/mod
GOOS := $(shell go env GOOS)

DOCKER_COMPOSE_FILE?=deployments/docker-compose.yml
DOCKER_COMPOSE_OVERRIDE?=deployments/docker-compose.override.yml

# define standard colors
# https://gist.github.com/rsperl/d2dfe88a520968fbc1f49db0a29345b9
ifneq (,$(findstring xterm,${TERM}))
	BLACK        := $(shell tput -Txterm setaf 0)
	RED          := $(shell tput -Txterm setaf 1)
	GREEN        := $(shell tput -Txterm setaf 2)
	YELLOW       := $(shell tput -Txterm setaf 3)
	LIGHTPURPLE  := $(shell tput -Txterm setaf 4)
	PURPLE       := $(shell tput -Txterm setaf 5)
	BLUE         := $(shell tput -Txterm setaf 6)
	WHITE        := $(shell tput -Txterm setaf 7)
	RESET        := $(shell tput -Txterm sgr0)
else
	BLACK        := ""
	RED          := ""
	GREEN        := ""
	YELLOW       := ""
	LIGHTPURPLE  := ""
	PURPLE       := ""
	BLUE         := ""
	WHITE        := ""
	RESET        := ""
endif

export


#
# TARGETS
#

all: info

.PHONY: info
info: 
	@echo -e "\n${GREEN} ${PROJECT_NAME} / Makefile ${RESET}\n"

	@echo -e "${YELLOW} make config  --- check dev environment ${RESET}"
	@echo -e "${YELLOW} make fmt     --- reformat the go source (gofmt) ${RESET}"
	@echo -e "${YELLOW} make docs    --- render documentation from code (go doc) ${RESET}\n"

	@echo -e "${YELLOW} make build   --- build project (docker image) ${RESET}"
	@echo -e "${YELLOW} make run     --- run project ${RESET}"
	@echo -e "${YELLOW} make logs    --- fetch container's logs ${RESET}"
	@echo -e "${YELLOW} make stop    --- stop and purge project (only docker containers!) ${RESET}"
	@echo -e ""

#
# deployment targets
#

.PHONY: dev
dev: fmt build run logs

.PHONY: prod
prod: run logs


#
# helper targets
#

.PHONY: fmt
fmt: version
	@echo -e "\n${YELLOW} Code reformating (gofmt)... ${RESET}\n"
	@gofmt -w -s .

.PHONY: config
config:
	@echo -e "\n${YELLOW} Running local configuration setup... ${RESET}\n"
	@go install github.com/swaggo/swag/cmd/swag@latest

.PHONY: docs
docs: 
	@echo -e "\n${YELLOW} Generating OpenAPI documentation... ${RESET}\n"
	@~/go/bin/swag init --parseDependency -ot json -g router.go --dir pkg/backend/ 
	@mv docs/swagger.json api/swagger.json
	@[ -f ${DOCKER_COMPOSE_OVERRIDE} ] \
		&& docker compose -f ${DOCKER_COMPOSE_FILE} -f ${DOCKER_COMPOSE_OVERRIDE} up litter-swagger -d --force-recreate \
		|| docker compose -f ${DOCKER_COMPOSE_FILE} up litter-swagger -d --force-recreate

.PHONY: build
build: 
	@echo -e "\n${YELLOW} Building the project (docker compose build)... ${RESET}\n"
	@[ -f ${DOCKER_COMPOSE_OVERRIDE} ] \
		&& DOCKER_BUILDKIT=1 docker compose -f ${DOCKER_COMPOSE_FILE} -f ${DOCKER_COMPOSE_OVERRIDE} build \
		|| DOCKER_BUILDKIT=1 docker compose -f ${DOCKER_COMPOSE_FILE} build

.PHONY: run
run:	
	@echo -e "\n${YELLOW} Starting project (docker compose up)... ${RESET}\n"
	@[ -f ${DOCKER_COMPOSE_OVERRIDE} ] \
		&& docker compose -f ${DOCKER_COMPOSE_FILE} -f ${DOCKER_COMPOSE_OVERRIDE} up --force-recreate --detach --remove-orphans \
		|| docker compose -f ${DOCKER_COMPOSE_FILE} up --force-recreate --detach --remove-orphans

.PHONY: logs
logs:
	@echo -e "\n${YELLOW} Fetching container's logs (CTRL-C to exit)... ${RESET}\n"
	@docker logs ${DOCKER_CONTAINER_NAME} -f

.PHONY: stop
stop:  
	@echo -e "\n${YELLOW} Stopping and purging project (docker compose down)... ${RESET}\n"
	@docker compose -f ${DOCKER_COMPOSE_FILE} down

.PHONY: version
version: 
	@[ -f "./.env" ] && cat .env | \
		sed -e 's/\(APP_PEPPER\)=\(.*\)/\1=xxx/' | \
		sed -e 's/\(REGISTRY\)=\(.*\)/\1=""/' | \
		sed -e 's/\(MAIL_SASL_USR\)=\(.*\)/\1=xxx/' | \
		sed -e 's/\(MAIL_SASL_PWD\)=\(.*\)/\1=xxx/' | \
		sed -e 's/\(MAIL_HOST\)=\(.*\)/\1=xxx/' | \
		sed -e 's/\(MAIL_PORT\)=\(.*\)/\1=xxx/' | \
		sed -e 's/\(GSC_TOKEN\)=\(.*\)/\1=xxx/' | \
		sed -e 's/\(GSC_URL\)=\(.*\)/\1=xxx/' | \
		sed -e 's/\(VAPID_PRIV_KEY\)=\(.*\)/\1=xxx/' | \
		sed -e 's/\(VAPID_PUB_KEY\)=\(.*\)/\1=xxx/' | \
		sed -e 's/\(VAPID_SUBSCRIBER\)=\(.*\)/\1=xxx/' | \
		sed -e 's/\(LOKI_URL\)=\(.*\)/\1=http:\/\/loki.example.com\/loki\/api\/v1\/push/' | \
		sed -e 's/\(APP_URLS_TRAEFIK\)=\(.*\)/\1=`littr.example.com`/' | \
		sed -e 's/\(API_TOKEN\)=\(.*\)/\1=xxx/' > .env.example && \
		sed -i 's/\/\/\(.*[[:blank:]]\)[0-9]*\.[0-9]*\.[0-9]*/\/\/\1${APP_VERSION}/' pkg/backend/router.go

.PHONY: push
push:
	@echo -e "\n${YELLOW} Pushing to git with tags... ${RESET}\n"
	@git tag -fa 'v${APP_VERSION}' -m 'v${APP_VERSION}'
	@git push --follow-tags
	
.PHONY: sh
sh:
	@echo -e "\n${YELLOW} Attaching container's (${DOCKER_CONTAINER_NAME})... ${RESET}\n"
	@docker exec -it ${DOCKER_CONTAINER_NAME} sh

.PHONY: flush
flush:
	@echo -e "\n${YELLOW} Flushing app data... ${RESET}\n"
	@docker cp test/data/polls.json ${DOCKER_CONTAINER_NAME}:/opt/data/polls.json
	@docker cp test/data/posts.json ${DOCKER_CONTAINER_NAME}:/opt/data/posts.json
	@docker cp test/data/users.json ${DOCKER_CONTAINER_NAME}:/opt/data/users.json
	@docker cp test/data/subscriptions.json ${DOCKER_CONTAINER_NAME}:/opt/data/subscriptions.json
	@docker cp test/data/tokens.json ${DOCKER_CONTAINER_NAME}:/opt/data/tokens.json

.PHONY: kill
kill:
	@echo -e "\n${YELLOW} Killing the container not to dump running caches... ${RESET}\n"
	@docker kill ${DOCKER_CONTAINER_NAME}

RUN_DATA_DIR=./.run_data
.PHONY: fetch_running_dump
fetch_running_dump:
	@echo -e "\n${YELLOW} Copying dumped data from the container... ${RESET}\n"
	@mkdir -p ${RUN_DATA_DIR}
	@docker cp ${DOCKER_CONTAINER_NAME}:/opt/data/users.json ${RUN_DATA_DIR}
	@docker cp ${DOCKER_CONTAINER_NAME}:/opt/data/polls.json ${RUN_DATA_DIR}
	@docker cp ${DOCKER_CONTAINER_NAME}:/opt/data/posts.json ${RUN_DATA_DIR}
	@docker cp ${DOCKER_CONTAINER_NAME}:/opt/data/tokens.json ${RUN_DATA_DIR}
	@docker cp ${DOCKER_CONTAINER_NAME}:/opt/data/subscriptions.json ${RUN_DATA_DIR}
	
.PHONY: backup
backup: fetch_running_dump
	@echo -e "\n${YELLOW} Making the backup archive... ${RESET}\n"
	@tar czvf /mnt/backup/litter-go/$(shell date +"%Y-%m-%d-%H:%M:%S").tar.gz ${RUN_DATA_DIR}

