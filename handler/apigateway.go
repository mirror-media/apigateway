package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"

	"github.com/gobwas/ws"
	"github.com/jensneuse/abstractlogger"
	"github.com/jensneuse/graphql-go-tools/pkg/execution"
	"github.com/jensneuse/graphql-go-tools/pkg/execution/datasource"
	"github.com/jensneuse/graphql-go-tools/pkg/graphql"
	gqhttp "github.com/jensneuse/graphql-go-tools/pkg/http"
	"github.com/sirupsen/logrus"
)

var graphqlDataSourceName = "graphql"

var logger = abstractlogger.NewLogrusLogger(logrus.New(), abstractlogger.DebugLevel)

var MemberQueryFields []string = []string{"__schema", "allMerchandises", "member", "merchandise"}
var MemberMutationFields []string = []string{"createmember", "updatemember", "createSubscriptionRecurring", "createsSubscriptionOneTime", "updatesubscription"}

func NewAPIGatewayGraphQLHandler() http.Handler {

	typeFieldConfigs := []datasource.TypeFieldConfiguration{}

	schemaString, err := os.ReadFile("/Users/chiu/dev/bcgodev/apigateway/graph/schema.graphqls")
	if err != nil {
		logrus.Panic(err)
	}
	schema, err := graphql.NewSchemaFromString(string(schemaString))
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

	buff := bytes.Buffer{}
	schema.IntrospectionResponse(&buff)

	// schemaJSON := buff.String()

	schemaConfig, _ := json.Marshal(datasource.SchemaDataSourcePlannerConfig{})

	// logrus.Info("schemaJSON:" + schemaJSON)

	graphqlTypeFieldConfigIntrospection := datasource.TypeFieldConfiguration{
		TypeName:  "query",
		FieldName: "__schema",
		DataSource: datasource.SourceConfig{
			Name:   "SchemaDataSource",
			Config: schemaConfig,
		},
	}

	typeFieldConfigs = append(typeFieldConfigs, graphqlTypeFieldConfigIntrospection)

	// FIXME Use config
	upstreamURL := "http://localhost:3000/api/graphql"

	memberQueryGraphqlTypeFieldConfigs := make([]datasource.TypeFieldConfiguration, len(MemberQueryFields))

	for _, f := range MemberQueryFields {
		memberQueryGraphqlTypeFieldConfigs = append(memberQueryGraphqlTypeFieldConfigs, datasource.TypeFieldConfiguration{
			TypeName:  "Query",
			FieldName: f,
			// Mapping: &datasource.MappingConfiguration{
			// 	Disabled: false,
			// 	Path:     "member",
			// },
			DataSource: datasource.SourceConfig{
				Name: graphqlDataSourceName,
				Config: jsonRawMessagify(map[string]interface{}{
					"url":    upstreamURL,
					"method": http.MethodPost,
				}),
			},
		})
	}

	typeFieldConfigs = append(typeFieldConfigs, memberQueryGraphqlTypeFieldConfigs...)

	memberMutationGraphqlTypeFieldConfigs := make([]datasource.TypeFieldConfiguration, len(MemberMutationFields))
	MemberMutationURL := "http://localhost:1234/v3/graphql/member"
	for _, f := range MemberMutationFields {
		memberMutationGraphqlTypeFieldConfigs = append(memberMutationGraphqlTypeFieldConfigs, datasource.TypeFieldConfiguration{
			TypeName:  "Mutation",
			FieldName: f,
			DataSource: datasource.SourceConfig{
				Name: graphqlDataSourceName,
				Config: jsonRawMessagify(map[string]interface{}{
					"url":    MemberMutationURL,
					"method": http.MethodPost,
				}),
			},
		})
	}

	typeFieldConfigs = append(typeFieldConfigs, memberMutationGraphqlTypeFieldConfigs...)

	plannerConfig := datasource.PlannerConfiguration{
		TypeFieldConfigurations: typeFieldConfigs,
	}
	basePlanner, err := datasource.NewBaseDataSourcePlanner([]byte(schemaString), plannerConfig, logger)
	if err != nil {
		logrus.Panic(err)
	}

	err = basePlanner.RegisterDataSourcePlannerFactory(graphqlDataSourceName, &datasource.GraphQLDataSourcePlannerFactoryFactory{})
	if err != nil {
		logrus.Panic(err)
	}
	err = basePlanner.RegisterDataSourcePlannerFactory("SchemaDataSource", datasource.SchemaDataSourcePlannerFactoryFactory{})
	if err != nil {
		logrus.Panic(err)
	}

	executionHandler := execution.NewHandler(basePlanner, nil)

	handler := gqhttp.NewGraphqlHTTPHandlerFunc(executionHandler, logger, &ws.DefaultHTTPUpgrader)

	return handler
}

func jsonRawMessagify(any interface{}) []byte {
	out, _ := json.Marshal(any)
	return out
}
