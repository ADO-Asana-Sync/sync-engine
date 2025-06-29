FROM golang:1.23-alpine AS builder

# hadolint ignore=DL3018
RUN apk add --no-cache \
	g++ \
	gcc \
	git

ARG VERSION=0.0.0-development
ARG COMMIT=none
ARG DATE=unknown

WORKDIR /github.com/ADO-Asana-Sync/sync-engine/
COPY ./go.mod .
COPY ./go.sum .
RUN go mod download
COPY . .
RUN mkdir /app && \
	CGO_ENABLED=1 GOOS=linux \
	go build \
	-ldflags "-s -w -X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.Date=${DATE}'" \
	-o /app/sync \
	./cmd/sync

FROM alpine:3
# hadolint ignore=DL3018
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/* ./
USER sync
ENV MONGO_URI=""
ENTRYPOINT [ "./sync" ]
EXPOSE 8080
