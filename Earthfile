FROM golang:1.17-alpine3.14
RUN apk --update --no-cache add git gcc musl-dev
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.42.1
ARG EARTHLY_GIT_SHORT_HASH
ARG EARTHLY_GIT_PROJECT_NAME
ARG PROJECT_NAME=udin2mqtt
ARG VERSION="0.0.0+${EARTHLY_GIT_SHORT_HASH}"
WORKDIR /${PROJECT_NAME}

deps:
    COPY go.mod go.sum ./
    RUN go mod download
    SAVE ARTIFACT go.mod AS LOCAL go.mod
    SAVE ARTIFACT go.sum AS LOCAL go.sum

build:
  FROM +deps
  COPY *.go .
  COPY pkg/ pkg/
  RUN CGO_ENABLED=0 go build \
      -ldflags "-s -w -X \"main.Version=${VERSION}\" ${LDFLAGS}" \
      -a -trimpath \
      -o build/${PROJECT_NAME}
  SAVE ARTIFACT build/${PROJECT_NAME} /${PROJECT_NAME} AS LOCAL build/${PROJECT_NAME}

lint:
  FROM +deps
  COPY .golangci.yml .
  COPY *.go .
  COPY pkg/ pkg/
  RUN CGO_ENABLED=0 go vet
  RUN golangci-lint run --tests=false

test:
  FROM +deps
  COPY *.go .
  COPY pkg/ pkg/
  COPY index.html pkg/ui/
  COPY static/ pkg/ui/static/
  RUN mkdir -p build
  RUN CGO_ENABLED=0 go test -coverprofile=build/coverage.out ./...
  SAVE ARTIFACT build/coverage.out AS LOCAL build/coverage.out

docker:
  BUILD +lint
  BUILD +test
  FROM scratch
  WORKDIR /${PROJECT_NAME}
  COPY +build/${PROJECT_NAME} .
  COPY index.html .
  COPY static/ static/
  ENTRYPOINT ["/${PROJECT_NAME}/${PROJECT_NAME}"]
  SAVE IMAGE ${PROJECT_NAME}:latest

all:
  BUILD +build
  BUILD +lint
  BUILD +test
  BUILD +docker
