package handler

import (
	"context"
	"net/http"
	"time"

	http2 "github.com/jensneuse/graphql-go-tools/examples/federation/gateway/http"
	"github.com/mirror-media/mm-apigateway/graph"

	"github.com/jensneuse/abstractlogger"
	"github.com/jensneuse/graphql-go-tools/pkg/engine/datasource/graphql_datasource"
	"github.com/jensneuse/graphql-go-tools/pkg/engine/datasource/httpclient"
	"github.com/jensneuse/graphql-go-tools/pkg/graphql"
	"github.com/jensneuse/graphql-go-tools/pkg/graphql/federation"
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

	defaultClient := httpclient.DefaultNetHttpClient

	factory := federation.NewEngineConfigV2Factory(
		&http.Client{
			Transport: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 30 * time.Second,
			},
			CheckRedirect: defaultClient.CheckRedirect,
			Jar:           defaultClient.Jar,
			Timeout:       defaultClient.Timeout,
		},
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
					URL: mutationUpstreamURL,
				},
				Federation: graphql_datasource.FederationConfiguration{
					Enabled:    false,
					ServiceSDL: string(mutationSchema.Document()),
				},
			},
		}...,
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

	// TODO extract the handler wrapper from the example
	handler := http2.NewGraphqlHTTPHandler(mergedSchema, executionEngine, nil, logger)

	return handler
}
