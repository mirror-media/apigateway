package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	gqlgenhendler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/machinebox/graphql"
	"github.com/mirror-media/apigateway/graph/member/mutationgraph"
	"github.com/mirror-media/apigateway/graph/member/mutationgraph/generated"
	"github.com/mirror-media/apigateway/handler"
	"github.com/mirror-media/apigateway/middleware"
	"github.com/mirror-media/apigateway/token"
	"golang.org/x/oauth2"

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
	v2tokenStateRouter := v2Router.Use(middleware.GetIDTokenOnly(server.firebaseClient))

	v2TokenAuthenticatedWithFirebaseRouter := v2tokenStateRouter.Use(middleware.AuthenticateIDToken(server.firebaseClient), middleware.GinContextToContextMiddleware(), middleware.FirebaseClientToContextMiddleware(server.firebaseClient), middleware.FirebaseDBClientToContextMiddleware(server.firebaseDatabaseClient))

	v2GraphHandler := handler.NewAPIGatewayGraphQLHandler("https://israfel.mirrormedia.mg/api/graphql", "http://localhost:8080/api/v2/graphql/member", "graphql/member/type.graphql", "graphql/member/query.graphql", "graphql/member/mutation.graphql")

	v2TokenAuthenticatedWithFirebaseRouter.POST("graphql/member", gin.WrapH(v2GraphHandler))

	// Public API
	// v1 api
	v1Router := apiRouter.Group("/v1")
	v1tokenStateRouter := v1Router.Use(middleware.GetIDTokenOnly(server.firebaseClient))
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

	// Private API
	// v1 User
	// It will save FirebaseClient and FirebaseDBClient to *gin.context, and *gin.context to *context
	// v1TokenAuthenticatedWithFirebaseRouter := v1Router.Use(middleware.AuthenticateIDToken(server.firebaseClient), middleware.GinContextToContextMiddleware(), middleware.FirebaseClientToContextMiddleware(server.firebaseClient), middleware.FirebaseDBClientToContextMiddleware(server.firebaseDatabaseClient))
	// svr := gqlgenhendler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{
	// 	Conf:       *server.Conf,
	// 	UserSvrURL: server.Conf.ServiceEndpoints.UserGraphQL,
	// 	// Token:      server.UserSvrToken,
	// 	// TODO Temp workaround
	// 	Client: func() *graphql.Client {
	// 		tokenString, err := server.UserSvrToken.GetTokenString()
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		src := oauth2.StaticTokenSource(
	// 			&oauth2.Token{
	// 				AccessToken: tokenString,
	// 				TokenType:   token.TypeJWT,
	// 			},
	// 		)
	// 		httpClient := oauth2.NewClient(context.Background(), src)
	// 		return graphql.NewClient(server.Services.UserGraphQL, graphql.WithHTTPClient(httpClient))
	// 	}(),
	// }}))
	// v1TokenAuthenticatedWithFirebaseRouter.POST("/graphql/user", gin.WrapH(svr))

	// v0 api proxy every request to the restful serverce
	v0Router := apiRouter.Group("/v0")
	v0tokenStateRouter := v0Router.Use(middleware.GetIDTokenOnly(server.firebaseClient), middleware.SetUserID(server.firebaseClient), middleware.GinContextToContextMiddleware())
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
	// FIXME proxied headers are in the request payload
	v2tokenStateRouter := v2Router.Use(middleware.GetIDTokenOnly(server.firebaseClient))

	svr := gqlgenhendler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &mutationgraph.Resolver{
		Conf:       *server.Conf,
		UserSvrURL: server.Conf.ServiceEndpoints.UserGraphQL,
		// Token:      server.UserSvrToken,
		// TODO Temp workaround
		Client: func() *graphql.Client {
			tokenString, err := server.UserSvrToken.GetTokenString()
			if err != nil {
				panic(err)
			}
			src := oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: tokenString,
					TokenType:   token.TypeJWT,
				},
			)
			httpClient := oauth2.NewClient(context.Background(), src)
			return graphql.NewClient(server.Services.UserGraphQL, graphql.WithHTTPClient(httpClient))
		}(),
	}}))
	v2tokenStateRouter.POST("/graphql/member", gin.WrapH(svr))

	return nil
}
