package server

import (
	"errors"
	"net/http"
	"net/url"

	gqlgenhendler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/jensneuse/graphql-go-tools/pkg/engine/datasource/httpclient"
	"github.com/machinebox/graphql"
	"github.com/mirror-media/apigateway/graph/member/mutationgraph"
	"github.com/mirror-media/apigateway/graph/member/mutationgraph/generated"
	"github.com/mirror-media/apigateway/handler"
	"github.com/mirror-media/apigateway/middleware"
	"github.com/mirror-media/apigateway/payment"
	"github.com/mirror-media/apigateway/token"

	"github.com/gin-gonic/gin"
)

type Reply struct {
	TokenState interface{} `json:"tokenState"`
	Data       interface{} `json:"data,omitempty"`
}

type Error struct {
	Message string `json:"message,omitempty"`
}
type ErrorReply struct {
	Errors []Error     `json:"errors,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

func SetHealthRoute(server *Server) error {

	if server.Conf == nil || server.FirebaseApp == nil {
		return errors.New("config or firebase app is nil")
	}

	router := server.Engine
	router.GET("/health", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusOK)
	})

	return nil
}

// SetRoute sets the routing for the gin engine
func SetRoute(server *Server) error {
	apiRouter := server.Engine.Group("/api")

	// v2 api
	v2Router := apiRouter.Group("/v2")
	v2tokenStateRouter := v2Router.Use(middleware.SetIDTokenOnly(server.firebaseClient))

	v2TokenAuthenticatedWithFirebaseRouter := v2tokenStateRouter.Use(middleware.AuthenticateIDToken(server.firebaseClient), middleware.FirebaseClientToContextMiddleware(server.firebaseClient), middleware.FirebaseDBClientToContextMiddleware(server.firebaseDatabaseClient))

	v2GraphHandler := handler.NewAPIGatewayGraphQLHandler("https://israfel.mirrormedia.mg/api/graphql", "http://localhost:8888/api/v2/graphql/member", "graph/member/type.graphql", "graph/member/query.graphql", "graph/member/mutation.graphql")

	v2GraphqlMemberRouter := v2TokenAuthenticatedWithFirebaseRouter.Use(middleware.AuthenticateMemberQueryAndFirebaseIDInArguments, middleware.PatchNullVariablesInGraphqlVariables)

	v2GraphqlMemberRouter.POST("graphql/member", gin.WrapH(v2GraphHandler))

	// v1 api
	v1Router := apiRouter.Group("/v1")
	v1tokenStateRouter := v1Router.Use(middleware.SetIDTokenOnly(server.firebaseClient))
	v1tokenStateRouter.GET("/tokenState", func(c *gin.Context) {
		t := c.Value(middleware.GCtxTokenKey).(token.Token)
		if t == nil {
			c.JSON(http.StatusBadRequest, Reply{
				TokenState: nil,
			})
			return
		}
		c.JSON(http.StatusOK, Reply{
			TokenState: t.GetTokenState(),
		})
	})

	// v0 api proxy every request to the restful serverce
	v0Router := apiRouter.Group("/v0")
	v0tokenStateRouter := v0Router.Use(middleware.SetIDTokenOnly(server.firebaseClient), middleware.SetUserID(server.firebaseClient))
	proxyURL, err := url.Parse(server.Conf.V0RESTfulSvrTargetURL)
	if err != nil {
		return err
	}

	v0tokenStateRouter.Any("/*wildcard", NewSingleHostReverseProxy(proxyURL, v0Router.BasePath(), server.Rdb, server.Conf.RedisService.Cache.TTL, server.Conf.ServiceEndpoints.UserGraphQL))

	return nil
}

func SetMemberMutationRoute(server *Server) error {

	apiRouter := server.Engine.Group("/api")

	// v2 api
	v2Router := apiRouter.Group("/v2")

	v2tokenStateRouter := v2Router.Use(middleware.SetIDTokenOnly(server.firebaseClient))

	v2TokenAuthenticatedWithFirebaseRouter := v2tokenStateRouter.Use(middleware.AuthenticateIDToken(server.firebaseClient), middleware.AuthenticateMemberQueryAndFirebaseIDInArguments, middleware.PatchNullVariablesInGraphqlVariables, middleware.FirebaseClientToContextMiddleware(server.firebaseClient), middleware.FirebaseDBClientToContextMiddleware(server.firebaseDatabaseClient))

	c := server.Conf

	svr := gqlgenhendler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &mutationgraph.Resolver{
		Conf:       *server.Conf,
		UserSvrURL: server.Conf.ServiceEndpoints.UserGraphQL,
		Client: func() *graphql.Client {
			httpClient := httpclient.DefaultNetHttpClient
			return graphql.NewClient(server.Services.UserGraphQL, graphql.WithHTTPClient(httpClient))
		}(),
		NewebpayStore: payment.NewebPayStore{
			CallbackHost:        c.NewebPayStore.CallbackHost,
			CallbackProtocol:    c.NewebPayStore.CallbackProtocol,
			ClientBackPath:      c.NewebPayStore.ClientBackPath,
			ID:                  c.NewebPayStore.ID,
			IsAbleToModifyEmail: payment.Boolean(c.NewebPayStore.IsAbleToModifyEmail),
			LoginType:           payment.NewebpayLoginType(c.NewebPayStore.LoginType),
			NotifyProtocol:      c.NewebPayStore.NotifyProtocol,
			NotifyHost:          c.NewebPayStore.NotifyHost,
			NotifyPath:          c.NewebPayStore.NotifyPath,
			Is3DSecure:          payment.Boolean(c.NewebPayStore.Is3DSecure),
			RespondType:         payment.NewebpayRespondType(c.NewebPayStore.RespondType),
			ReturnPath:          c.NewebPayStore.ReturnPath,
			Version:             c.NewebPayStore.Version,
		},
	}}))
	v2TokenAuthenticatedWithFirebaseRouter.POST("/graphql/member", gin.WrapH(svr))

	return nil
}
