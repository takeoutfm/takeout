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

SOURCES = $(wildcard *.go internal/*/*.go model/*.go spiff/*.go view/*.go)

CLIENT_SOURCES = $(wildcard client/*.go model/*.go spiff/*.go view/*.go)

PLAYER_SOURCES = $(wildcard player/*.go)

RES_DIR = internal/server/res
RES_STATIC = $(wildcard ${RES_DIR}/static/*.css ${RES_DIR}/static/*.html \
	${RES_DIR}/static/*.js ${RES_DIR}/static/*.svg)
RES_ROOT = $(wildcard ${RES_DIR}/template/*.html)
RES_MUSIC = $(wildcard ${RES_DIR}/template/music/*.html)
RES_PODCAST = $(wildcard ${RES_DIR}/template/podcast/*.html)
RES_VIDEO = $(wildcard ${RES_DIR}/template/video/*.html)
RES_TEMPLATE = ${RES_ROOT} ${RES_MUSIC} ${RES_PODCAST} ${RES_VIDEO}
RESOURCES = ${RES_STATIC} ${RES_TEMPLATE}

#
TAKEOUT_CMD_DIR = cmd/takeout
TAKEOUT_CMD_TARGET = ${TAKEOUT_CMD_DIR}/takeout
TAKEOUT_CMD_SRC = $(wildcard ${TAKEOUT_CMD_DIR}/*.go)

#
PLAYOUT_CMD_DIR = cmd/playout
PLAYOUT_CMD_TARGET = ${PLAYOUT_CMD_DIR}/playout
PLAYOUT_CMD_SRC = $(wildcard ${PLAYOUT_CMD_DIR}/*.go)

#
TMDB_CMD_DIR = tools/cmd/tmdb
TMDB_CMD_TARGET = ${TMDB_CMD_DIR}/tmdb
TMDB_CMD_SRC = $(wildcard ${TMDB_CMD_DIR}/*.go)

.PHONY: all install clean

all: takeout playout

takeout: ${TAKEOUT_CMD_TARGET}

${TAKEOUT_CMD_TARGET}: ${TAKEOUT_CMD_SRC} ${SOURCES} ${RESOURCES}
	@cd ${TAKEOUT_CMD_DIR} && ${GO} build

install-takeout: takeout
	@cd ${TAKEOUT_CMD_DIR} && ${GO} install

playout: ${PLAYOUT_CMD_TARGET}

${PLAYOUT_CMD_TARGET}: ${PLAYOUT_CMD_SRC} ${SOURCES} ${CLIENT_SOURCES} ${PLAYER_SOURCES}
	@cd ${PLAYOUT_CMD_DIR} && ${GO} build

install-playout: playout
	@cd ${PLAYOUT_CMD_DIR} && ${GO} install

tmdb: ${TMDB_CMD_TARGET}

${TMDB_CMD_TARGET}: ${TMDB_CMD_SRC} ${SOURCES}
	@cd ${TMDB_CMD_DIR} && ${GO} build

install: install-takeout install-playout

clean:
	@cd ${TAKEOUT_CMD_DIR} && ${GO} clean
	rm -f ${TAKEOUT_CMD_TARGET}
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
