package handler

import (
	"net/http"

	"github.com/jensneuse/graphql-go-tools/pkg/playground"
	"github.com/sirupsen/logrus"
)

func NewPlaygroundHandelr() http.Handler {
	// playground

	playgroundConfig := playground.Config{
		PathPrefix:          "/playground",
		PlaygroundPath:      "/playground/files",
		GraphqlEndpointPath: "/graphql",
		// GraphQLSubscriptionEndpointPath: "/graphqlws",
	}

	p := playground.New(playgroundConfig)
	handlers, err := p.Handlers()
	if err != nil {
		logrus.Panic(err)
	}
	return handlers[0].Handler

	// engine.Any("/playground", gin.WrapH(handlers[0].Handler))
	// engine.Static("/playground/files", "/Users/chiu/dev/bcgodev/apigateway/gateway-poc/playground/files")
}
