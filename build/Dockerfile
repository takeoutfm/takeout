# Copyright 2023 defsub
#
# This file is part of TakeoutFM.
#
# TakeoutFM is free software: you can redistribute it and/or modify it under the
# terms of the GNU Affero General Public License as published by the Free
# Software Foundation, either version 3 of the License, or (at your option)
# any later version.
#
# TakeoutFM is distributed in the hope that it will be useful, but WITHOUT ANY
# WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
# FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
# more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with TakeoutFM.  If not, see <https://www.gnu.org/licenses/>.

# This docker build use multi-stage builds to first build takeout using the
# full golang image and then copies the result to minimal debian image.

# build stage
FROM golang:1.19.2-bullseye as builder
ARG src=/go/src/takeoutfm.dev/takeout

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading
# them in subsequent builds if they change
WORKDIR $src/
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN make install-server

# final stage
FROM debian:bullseye-slim
ARG src=/go/src/takeoutfm.dev/takeout
ARG user=takeout
ARG group=$user
ARG etc=/etc/$user/
ARG home=/var/lib/$user
ARG uid=1023
ARG gid=1023

RUN groupadd -g $gid $group && useradd -r -d $home -m -g $group -u $uid $user
COPY --from=builder $src/doc/takeout.yaml $etc
COPY --from=builder /go/bin/takeout /go/bin/takeout

# go home
WORKDIR $home
USER $user
CMD ["/go/bin/takeout", "run"]

# sudo docker run -name takeout -p 3000:3000 -v /home/defsub/takeout:/var/lib/takeout takeout:latest /bin/bash
