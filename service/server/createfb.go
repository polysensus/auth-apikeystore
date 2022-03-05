package server

import (
	"context"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/robinbryce/apikeystore/apibin"
	"github.com/robinbryce/apikeystore/service/keys"
)

func (a *APIKeyCreator) Create(
	ctx context.Context, in *apibin.CreateRequest) (*flatbuffers.Builder, error) {

	apikey, err := a.create(
		ctx, string(in.Audience()), string(in.Scopes()),
		keys.WithDisplayName(string(in.DisplayName())),
	)
	if err != nil {
		return nil, err
	}

	b := flatbuffers.NewBuilder(0)
	bapikey := b.CreateString(apikey)
	apibin.CreateResultStart(b)
	apibin.CreateResultAddApikey(b, bapikey)
	b.Finish(apibin.CreateResultEnd(b))
	return b, nil
}
