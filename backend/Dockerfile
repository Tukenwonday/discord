# Multi stage build: compile a static binary, then ship it on a minimal image
# that only carries the certificate store and the timezone database.
# The build stage needs Go 1.23+ because github.com/livekit/protocol declares
# "go 1.23"; the official images pin GOTOOLCHAIN=local so the toolchain cannot
# upgrade itself and the base image has to match.
FROM golang:1.23-alpine AS build

# github.com/livekit/server-sdk-go resolves its own module through VCS and
# shells out to git, which the alpine golang image does not ship. Without it the
# build fails with: exec: "git": executable file not found in $PATH.
RUN apk add --no-cache git

WORKDIR /src

# go.mod here lists direct dependencies only, and go.sum is generated rather than
# committed, so a bare `go mod download` can fail on a fresh clone. Run `go mod
# tidy` first: it resolves indirect requirements and writes go.sum inside the
# image, which keeps the build reproducible without extra manual steps.
COPY go.mod ./
COPY . .

RUN go mod tidy \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cordis ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S cordis \
    && adduser -S -G cordis -h /app cordis \
    && mkdir -p /app/data/uploads \
    && chown -R cordis:cordis /app

ENV APP_ENV=production \
    GIN_MODE=release \
    UPLOAD_DIR=/app/data/uploads

WORKDIR /app

COPY --from=build /out/cordis /usr/local/bin/cordis

USER cordis

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/cordis"]