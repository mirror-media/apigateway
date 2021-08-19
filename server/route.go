package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/machinebox/graphql"
	"github.com/mirror-media/mm-apigateway/middleware"
	"github.com/mirror-media/mm-apigateway/token"
	"golang.org/x/oauth2"

	"github.com/gin-gonic/gin"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/mirror-media/mm-apigateway/graph"
	"github.com/mirror-media/mm-apigateway/graph/generated"
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
	v1TokenAuthenticatedWithFirebaseRouter := v1Router.Use(middleware.AuthenticateIDToken(server.firebaseClient), middleware.GinContextToContextMiddleware(), middleware.FirebaseClientToContextMiddleware(server.firebaseClient), middleware.FirebaseDBClientToContextMiddleware(server.firebaseDatabaseClient))
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{
		Conf:       *server.Conf,
		UserSrvURL: server.Conf.ServiceEndpoints.UserGraphQL,
		// Token:      server.UserSrvToken,
		// TODO Temp workaround
		Client: func() *graphql.Client {
			tokenString, err := server.UserSrvToken.GetTokenString()
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
	v1TokenAuthenticatedWithFirebaseRouter.POST("/graphql/user", gin.WrapH(srv))

	// v0 api proxy every request to the restful serverce
	v0Router := apiRouter.Group("/v0")
	v0tokenStateRouter := v0Router.Use(middleware.GetIDTokenOnly(server.firebaseClient))
	proxyURL, err := url.Parse(server.Conf.V0RESTfulSrvTargetURL)
	if err != nil {
		return err
	}

	v0tokenStateRouter.Any("/*wildcard", NewSingleHostReverseProxy(proxyURL, v0Router.BasePath(), server.Rdb, server.Conf.RedisService.Cache.TTL))

	return nil
}
