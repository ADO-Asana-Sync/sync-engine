package asana

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"golang.org/x/oauth2"
)

type Asana struct {
	Client *http.Client
}

func (a *Asana) Connect(ctx context.Context, pat string) {
	tracer := otel.GetTracerProvider().Tracer("")
	_, span := tracer.Start(ctx, "asana.Connect")
	defer span.End()

	asanaCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tok := &oauth2.Token{AccessToken: pat}
	conf := &oauth2.Config{}
	a.Client = conf.Client(asanaCtx, tok)
}
