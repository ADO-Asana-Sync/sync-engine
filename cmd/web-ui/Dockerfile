FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc g++ git

ARG VERSION=0.0.0-development
ARG COMMIT=none
ARG DATE=unknown

WORKDIR /github.com/ADO-Asana-Sync/sync-engine/
COPY ./go.mod .
COPY ./go.sum .
RUN go mod download
COPY . .
RUN mkdir /app
# Get the latest commit hash and date if they are not supplied
RUN if [ "${COMMIT}" = "none" ]; then \
	COMMIT=$(git rev-parse HEAD); \
	fi && \
	if [ "${DATE}" = "unknown" ]; then \
	DATE=$(git show -s --format=%ci HEAD); \
	fi
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-s -w -X 'github.com/ADO-Asana-Sync/sync-engine/cmd/web-ui/main.Version=${VERSION}' -X 'github.com/ADO-Asana-Sync/sync-engine/cmd/web-ui/main.Commit=${COMMIT}' -X 'github.com/ADO-Asana-Sync/sync-engine/cmd/web-ui/main.Date=${DATE}'" -o /app/web-ui ./cmd/web-ui

FROM alpine:3
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/* ./
COPY --from=builder /github.com/ADO-Asana-Sync/sync-engine/cmd/web-ui/static ./static
COPY --from=builder /github.com/ADO-Asana-Sync/sync-engine/cmd/web-ui/templates ./templates
USER sync
ENV MONGO_URI=""
ENTRYPOINT [ "./web-ui" ]
EXPOSE 8080
