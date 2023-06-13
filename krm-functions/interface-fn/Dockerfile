#  Copyright 2023 The Nephio Authors.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

FROM golang:1.20-alpine
ENV CGO_ENABLED=0
WORKDIR /go/src/
COPY . .
WORKDIR /go/src/interface-fn
RUN go install
RUN go build -o /usr/local/bin/function ./

FROM gcr.io/distroless/static:latest
COPY --from=0 /usr/local/bin/function /usr/local/bin/function
ENTRYPOINT ["function"]
