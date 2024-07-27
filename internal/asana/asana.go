package asana

import (
	"context"
	"net/http"
	"time"

	"github.com/ADO-Asana-Sync/sync-engine/internal/helpers"

	"golang.org/x/oauth2"
)

type Asana struct {
	Client *http.Client
}

func (a *Asana) Connect(ctx context.Context, pat string) {
	ctx, span := helpers.StartSpanOnTracerFromContext(ctx, "asana.Connect")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	tok := &oauth2.Token{AccessToken: pat}
	conf := &oauth2.Config{}
	a.Client = conf.Client(ctx, tok)
}
