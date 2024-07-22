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
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-s -w -X 'github.com/ADO-Asana-Sync/sync-engine/cmd/sync/main.version=$VERSION' -X 'github.com/ADO-Asana-Sync/sync-engine/cmd/sync/main.commit=$COMMIT' -X 'github.com/ADO-Asana-Sync/sync-engine/cmd/sync/main.date=$DATE'" -o /app/sync ./cmd/sync

FROM alpine:3
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/* ./
USER sync
ENV MONGO_URI=""
ENTRYPOINT [ "./sync" ]
EXPOSE 8080
