package handler

import (
	"context"
	"net/http"
	"os"
	"time"

	http2 "github.com/jensneuse/graphql-go-tools/examples/federation/gateway/http"

	"github.com/jensneuse/abstractlogger"
	"github.com/jensneuse/graphql-go-tools/pkg/engine/datasource/graphql_datasource"
	"github.com/jensneuse/graphql-go-tools/pkg/engine/datasource/httpclient"
	"github.com/jensneuse/graphql-go-tools/pkg/graphql"
	"github.com/jensneuse/graphql-go-tools/pkg/graphql/federation"
	"github.com/sirupsen/logrus"
)

var logger = abstractlogger.NewLogrusLogger(logrus.New(), abstractlogger.InfoLevel)

func NewAPIGatewayGraphQLHandler(memberUpstreamURL, schemaPath string) http.Handler {
	schemaReader, err := os.Open(schemaPath)
	if err != nil {
		logrus.Panic(err)
	}

	schema, err := graphql.NewSchemaFromReader(schemaReader)
	schemaReader.Close()
	if err != nil {
		logrus.Panic(err)
	}
	validation, err := schema.Validate()
	if err != nil {
		logrus.Panic(err)
	}
	if !validation.Valid {
		validation.Errors.ErrorByIndex(0)
		logrus.Panic("schema is not valid:", validation.Errors.Error(), "first one is:", validation.Errors.ErrorByIndex(0))
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
		graphql_datasource.Configuration{
			Fetch: graphql_datasource.FetchConfiguration{
				URL: memberUpstreamURL,
			},
			Federation: graphql_datasource.FederationConfiguration{
				Enabled:    false,
				ServiceSDL: string(schema.Document()),
			},
		},
	)

	engineConfig, err := factory.EngineV2Configuration()
	if err != nil {
		logrus.Panic(err)
	}

	executionEngine, _ := graphql.NewExecutionEngineV2(context.TODO(), logger, engineConfig)

	// TODO extract the handler wrapper from the example
	handler := http2.NewGraphqlHTTPHandler(schema, executionEngine, nil, logger)

	return handler
}
