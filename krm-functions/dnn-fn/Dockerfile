#  Copyright 2023-2025 The Nephio Authors.
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

FROM golang:1.23.5-alpine@sha256:47d337594bd9e667d35514b241569f95fb6d95727c24b19468813d596d5ae596
ENV CGO_ENABLED=0
WORKDIR /go/src/
COPY krm-functions/ krm-functions/
WORKDIR /go/src/krm-functions/dnn-fn
RUN go install
RUN go build -o /usr/local/bin/function ./

FROM gcr.io/distroless/static@sha256:3f2b64ef97bd285e36132c684e6b2ae8f2723293d09aae046196cca64251acac
COPY --from=0 /usr/local/bin/function /usr/local/bin/function
ENTRYPOINT ["function"]
