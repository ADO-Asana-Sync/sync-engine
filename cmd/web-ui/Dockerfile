FROM golang:1.23-alpine AS builder

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
RUN mkdir /app
RUN CGO_ENABLED=1 GOOS=linux \
	go build \
	-ldflags "-s -w -X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.Date=${DATE}'" \
	-o /app/web-ui \
	./cmd/web-ui

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
