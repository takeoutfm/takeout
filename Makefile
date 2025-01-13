# Copyright 2023 defsub
#
# This file is part of Takeout.
#
# Takeout is free software: you can redistribute it and/or modify it under the
# terms of the GNU Affero General Public License as published by the Free
# Software Foundation, either version 3 of the License, or (at your option)
# any later version.
#
# Takeout is distributed in the hope that it will be useful, but WITHOUT ANY
# WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
# FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
# more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with Takeout.  If not, see <https://www.gnu.org/licenses/>.

GO = go
DOCKER = sudo docker

DOCKER_USER ?= defsub
DOCKER_IMAGE ?= takeout

GIT_VERSION ?= $(shell git log --format="%h" -n 1)

TAKEOUT_VERSION = $(shell sed -n -e 's/\s*Version = "\(.*\)"\s*/\1/p' takeout.go)

COMMON_SOURCES = $(wildcard *.go lib/*/*.go model/*.go spiff/*.go view/*.go)

INTERNAL_SOURCES = $(wildcard internal/*/*.go)

SERVER_SOURCES = ${INTERNAL_SOURCES} ${COMMON_SOURCES}

PLAYOUT_SOURCES = $(wildcard client/*.go player/*.go) ${INTERNAL_SOURCES} ${COMMON_SOURCES}

# server embedded resources
RES_DIR = internal/server/res
RES_STATIC = $(wildcard ${RES_DIR}/static/*.css ${RES_DIR}/static/*.html \
	${RES_DIR}/static/*.js ${RES_DIR}/static/*.svg)
RES_ROOT = $(wildcard ${RES_DIR}/template/*.html)
RES_MUSIC = $(wildcard ${RES_DIR}/template/music/*.html)
RES_PODCAST = $(wildcard ${RES_DIR}/template/podcast/*.html)
RES_FILM = $(wildcard ${RES_DIR}/template/film/*.html)
RES_TV = $(wildcard ${RES_DIR}/template/tv/*.html)
RES_TEMPLATE = ${RES_ROOT} ${RES_MUSIC} ${RES_PODCAST} ${RES_FILM} ${RES_TV}
SERVER_RESOURCES = ${RES_STATIC} ${RES_TEMPLATE}

# server
SERVER_CMD_DIR = cmd/takeout
SERVER_CMD_TARGET = ${SERVER_CMD_DIR}/takeout
SERVER_CMD_SRC = $(wildcard ${SERVER_CMD_DIR}/*.go)

# playout
PLAYOUT_CMD_DIR = cmd/playout
PLAYOUT_CMD_TARGET = ${PLAYOUT_CMD_DIR}/playout
PLAYOUT_CMD_SRC = $(wildcard ${PLAYOUT_CMD_DIR}/*.go)

# tmdb
TMDB_CMD_DIR = tools/cmd/tmdb
TMDB_CMD_TARGET = ${TMDB_CMD_DIR}/tmdb
TMDB_CMD_SRC = $(wildcard ${TMDB_CMD_DIR}/*.go)

.PHONY: all install clean

all: server playout

#
server: ${SERVER_CMD_TARGET}

${SERVER_CMD_TARGET}: ${SERVER_CMD_SRC} ${SERVER_SOURCES} ${SERVER_RESOURCES}
	@cd ${SERVER_CMD_DIR} && ${GO} build

install-server: server
	@cd ${SERVER_CMD_DIR} && ${GO} install

#
playout: ${PLAYOUT_CMD_TARGET}

${PLAYOUT_CMD_TARGET}: ${PLAYOUT_CMD_SRC} ${PLAYOUT_SOURCES}
	@cd ${PLAYOUT_CMD_DIR} && ${GO} build

install-playout: playout
	@cd ${PLAYOUT_CMD_DIR} && ${GO} install

#
tmdb: ${TMDB_CMD_TARGET}

${TMDB_CMD_TARGET}: ${TMDB_CMD_SRC} ${COMMON_SOURCES}
	@cd ${TMDB_CMD_DIR} && ${GO} build

test:
	${GO} test ./...

test-coverage:
	-${GO} test -coverprofile cover.out ./...
	${GO} tool cover -func=cover.out

install: install-server install-playout

clean:
	@cd ${SERVER_CMD_DIR} && ${GO} clean
	rm -f ${SERVER_CMD_TARGET}
	@cd ${PLAYOUT_CMD_DIR} && ${GO} clean
	rm -f ${PLAYOUT_CMD_TARGET}
	@cd ${TMDB_CMD_DIR} && ${GO} clean
	rm -f ${TMDB_CMD_TARGET}

docker docker-build: clean
	${DOCKER} build --rm -t ${DOCKER_IMAGE} build

docker-push:
	${DOCKER} tag ${DOCKER_IMAGE} ${DOCKER_USER}/${DOCKER_IMAGE}:latest
	${DOCKER} tag ${DOCKER_IMAGE} ${DOCKER_USER}/${DOCKER_IMAGE}:${GIT_VERSION}
	${DOCKER} push ${DOCKER_USER}/${DOCKER_IMAGE}:latest
	${DOCKER} push ${DOCKER_USER}/${DOCKER_IMAGE}:${GIT_VERSION}

# manually update version in takeout.go
# manually git commit -a
# push will add git version tag as needed, push origin, push tag
push:
	@git tag --list | grep -q v${TAKEOUT_VERSION} || git tag v${TAKEOUT_VERSION}
	@git push origin
	@git push origin v${TAKEOUT_VERSION}
