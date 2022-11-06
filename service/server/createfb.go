package server

import (
	"context"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/polysensus/auth-apikeystore/apibin"
)

func (a *APIKeyCreator) Create(
	ctx context.Context, in *apibin.CreateRequest) (*flatbuffers.Builder, error) {

	_, apikey, err := a.create(
		ctx, string(in.DisplayName()), string(in.Audience()), string(in.Scopes()))
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
