ARG VERSION=1.24.5

# Build Stage
FROM golang:${VERSION}-alpine AS builder
# hadolint ignore=DL3018
RUN apk --no-cache add make git gcc libtool musl-dev ca-certificates dumb-init

ARG BRANCH
ARG	BUILD_TIME
ARG BUILD_URL
ARG	COMMIT_ID
ARG APP_VERSION

WORKDIR /app

# Copy the source code
COPY ./ ./

RUN go mod download && go mod tidy

WORKDIR /app/cmd/go-web-layout

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s \
    -X github.com/manuelarte/go-web-layout/internal/info.Branch=${BRANCH} \
    -X github.com/manuelarte/go-web-layout/internal/info.BuildTime=${BUILD_TIME} \
    -X github.com/manuelarte/go-web-layout/internal/info.BuildURL=${BUILD_URL} \
    -X github.com/manuelarte/go-web-layout/internal/info.CommitID=${COMMIT_ID} \
    -X github.com/manuelarte/go-web-layout/internal/info.Version=${APP_VERSION}"  \
    -o /go-web-layout

# Final Stage
FROM alpine:3
# hadolint ignore=DL3018
RUN apk --no-cache add ca-certificates dumb-init

# Copy the binary from builder stage
COPY --from=builder /go-web-layout /usr/local/bin/go-web-layout

EXPOSE 3001

# Run
ENTRYPOINT ["/usr/local/bin/go-web-layout"]
