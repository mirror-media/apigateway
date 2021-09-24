package handler

import (
	"context"
	"net/http"

	"github.com/mirror-media/apigateway/graph"
	"github.com/mirror-media/apigateway/graph/http2"

	"github.com/jensneuse/abstractlogger"
	"github.com/jensneuse/graphql-go-tools/pkg/engine/datasource/graphql_datasource"
	"github.com/jensneuse/graphql-go-tools/pkg/graphql"
	"github.com/sirupsen/logrus"
)

var logger = abstractlogger.NewLogrusLogger(logrus.New(), abstractlogger.InfoLevel)

func NewAPIGatewayGraphQLHandler(memberUpstreamURL, mutationUpstreamURL, typeSchemaPath, querySchemaPath, mutationSchemaPath string) http.Handler {

	querySchema, err := graph.AlchemizeSchema(typeSchemaPath, querySchemaPath)
	if err != nil {
		logrus.Panic(err)
	}

	validation, err := querySchema.Validate()
	if err != nil {
		logrus.Panic(err)
	}
	if !validation.Valid {
		validation.Errors.ErrorByIndex(0)
		logrus.Panic("query schema is not valid:", validation.Errors.Error(), "first one is:", validation.Errors.ErrorByIndex(0))
	}

	mutationSchema, err := graph.AlchemizeSchema(typeSchemaPath, mutationSchemaPath)
	if err != nil {
		logrus.Panic(err)
	}

	validation, err = mutationSchema.Validate()
	if err != nil {
		logrus.Panic(err)
	}
	if !validation.Valid {
		validation.Errors.ErrorByIndex(0)
		logrus.Panic("mutation schema is not valid:", validation.Errors.Error(), "first one is:", validation.Errors.ErrorByIndex(0))
	}

	factory := graphql.NewFederationEngineConfigFactory(
		[]graphql_datasource.Configuration{
			{
				Fetch: graphql_datasource.FetchConfiguration{
					URL: memberUpstreamURL,
				},
				Federation: graphql_datasource.FederationConfiguration{
					Enabled:    false,
					ServiceSDL: string(querySchema.Document()),
				},
			}, {
				Fetch: graphql_datasource.FetchConfiguration{
					URL:    mutationUpstreamURL,
					Header: http.Header{"Authorization": []string{"{{ .request.headers.Authorization }}"}},
				},
				Federation: graphql_datasource.FederationConfiguration{
					Enabled:    false,
					ServiceSDL: string(mutationSchema.Document()),
				},
			},
		},
	)

	engineConfig, err := factory.EngineV2Configuration()
	if err != nil {
		logrus.Panic(err)
	}

	executionEngine, _ := graphql.NewExecutionEngineV2(context.TODO(), logger, engineConfig)

	mergedSchema, err := factory.MergedSchema()
	if err != nil {
		logrus.Panic(err)
	}

	handler := http2.NewGraphqlHTTPHandler(mergedSchema, executionEngine, nil, logger)

	return handler
}
