# syntax=docker/dockerfile:1
#
# litter-go / Dockerfile
#
 

#
# stage 0 -- build
#

# https://hub.docker.com/_/golang
ARG GOLANG_VERSION
FROM --platform=linux/amd64 golang:${GOLANG_VERSION}-alpine3.19 AS litter-build

ARG APP_NAME
ARG APP_PEPPER
ARG APP_VERSION
ARG API_TOKEN
ARG VAPID_PUB_KEY

ENV APP_NAME ${APP_NAME}
ENV APP_PEPPER ${APP_PEPPER}
ENV APP_VERSION ${APP_VERSION}
ENV API_TOKEN ${API_TOKEN}
ENV CGO_ENABLED 0
ENV VAPID_PUB_KEY ${VAPID_PUB_KEY}

RUN --mount=type=cache,target=/var/cache/apk \
	apk add git gcc build-base libc-dev

WORKDIR /go/src/${APP_NAME}
COPY go.mod .

ARG GOMODCACHE GOCACHE
RUN --mount=type=cache,target="$GOMODCACHE" go mod download

COPY . .

# build the client -- wasm binary
#RUN GOARCH=wasm GOOS=js go build -o web/app.wasm -tags wasm -ldflags "-X 'go.savla.dev/littr/config.APIToken=$API_TOKEN' -X 'go.savla.dev/littr/config.Version=$APP_VERSION' -X 'go.savla.dev/littr/config.Pepper=$APP_PEPPER'"
RUN --mount=type=cache,target="$GOMODCACHE" \
	--mount=type=cache,target="$GOCACHE" \
	GOARCH=wasm GOOS=js go build -o web/app.wasm -tags wasm -ldflags "-X 'go.savla.dev/littr/frontend.appVersion=$APP_VERSION' -X 'go.savla.dev/littr/frontend.vapidPublicKey=$VAPID_PUB_KEY'"
#RUN GOARCH=wasm GOOS=js go build -o web/app.wasm -tags wasm 

# build the server
#RUN go build -ldflags "-X 'litter-go/config.Version=$APP_VERSION'" ${APP_NAME}
RUN --mount=type=cache,target="$GOMODCACHE" \
	--mount=type=cache,target="$GOCACHE" \
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o littr


#
# stage 1 -- release
#

FROM alpine:3.19 AS litter-release

ARG APP_FLAGS
ARG APP_PEPPER
ARG APP_VERSION
ARG DOCKER_INTERNAL_PORT
ARG DOCKER_USER

ENV APP_FLAGS ${APP_FLAGS}
ENV APP_PEPPER ${APP_PEPPER}
ENV APP_VERSION ${APP_VERSION}
ENV DOCKER_DEV_PORT ${DOCKER_INTERNAL_PORT}
ENV DOCKER_USER ${DOCKER_USER}

RUN apk update && apk add tzdata

RUN adduser -D -h /opt -s /bin/sh ${DOCKER_USER}

COPY web/ /opt/web/
COPY --chown=1000:1000 --chmod=750 data/ /opt/data/
COPY --chown=1000:1000 --chmod=750 data/.gitkeep /opt/pix/
#COPY .script/periodic-dump.sh /opt/periodic-dump.sh
COPY --from=litter-build /go/src/litter-go/littr /opt/littr
COPY --from=litter-build /go/src/litter-go/web/app.wasm /opt/web/app.wasm

# workaround for pix
RUN cd /opt/web && ln -s ../pix .
RUN chown -R ${DOCKER_USER}:${DOCKER_USER} /opt/

WORKDIR /opt
USER ${DOCKER_USER}
EXPOSE ${DOCKER_INTERNAL_PORT}
ENTRYPOINT ["/opt/littr"]
