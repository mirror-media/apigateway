package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/machinebox/graphql"
	"github.com/mirror-media/mm-apigateway/middleware"
	"github.com/mirror-media/mm-apigateway/token"
	"github.com/tidwall/sjson"
	"golang.org/x/oauth2"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/mirror-media/mm-apigateway/graph"
	"github.com/mirror-media/mm-apigateway/graph/generated"
)

// TODO remove me and use the state from Israfil only
const MemberStateNone = "none"

// GetIDTokenOnly is a middleware to construct the token.Token interface
func GetIDTokenOnly(server *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := log.WithFields(log.Fields{
			"path": c.FullPath(),
		})
		// Create a Token Instance
		authHeader := c.GetHeader("Authorization")
		firebaseClient := server.FirebaseClient
		token, err := token.NewFirebaseToken(authHeader, firebaseClient)
		if err != nil {
			logger.Info(err)
			c.Next()
			return
		}
		c.Set(middleware.GCtxTokenKey, token)
		c.Next()
	}
}

// AuthenticateIDToken is a middleware to authenticate the request and save the result to the context
func AuthenticateIDToken(server *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := log.WithFields(log.Fields{
			"path": c.FullPath(),
		})
		// Create a Token Instance
		t := c.Value(middleware.GCtxTokenKey)
		if t == nil {
			err := errors.New("no token provided")
			logger.Info(err)
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorReply{
				Errors: []Error{{Message: err.Error()}},
			})
			return
		}
		tt := t.(token.Token)

		if tt.GetTokenState() != token.OK {
			logger.Info(tt.GetTokenState())
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorReply{
				Errors: []Error{{Message: tt.GetTokenState()}},
			})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Because GetTokenState() already fetch the public key and cache it. Here VerifyIDToken() would only verify the signature.
		firebaseClient := server.FirebaseClient
		tokenString, _ := tt.GetTokenString()
		idToken, err := firebaseClient.VerifyIDToken(ctx, tokenString)
		if err != nil {
			logger.Info(err.Error())
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorReply{
				Errors: []Error{{Message: err.Error()}},
			})
			return
		}
		c.Set(middleware.GCtxUserIDKey, idToken.Subject)
		c.Next()
	}
}

func GinContextToContextMiddleware(server *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), middleware.CtxGinContexKey, c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func FirebaseClientToContextMiddleware(server *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), middleware.CtxFirebaseClientKey, server.FirebaseClient)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func FirebaseDBClientToContextMiddleware(server *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), middleware.CtxFirebaseDatabaseClientKey, server.FirebaseDatabaseClient)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func singleJoiningSlash(a, b string) string {

	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")

	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}

	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()
	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")
	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}

	return a.Path + b.Path, apath + bpath

}

func ModifyReverseProxyResponse(c *gin.Context, rdb Rediser, cacheTTL int) func(*http.Response) error {
	logger := log.WithFields(log.Fields{
		"path": c.FullPath(),
	})
	return func(r *http.Response) error {
		body, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			logger.Errorf("encounter error when reading proxy response:", err)
			return err
		}

		var tokenState string

		tokenSaved, exist := c.Get(middleware.GCtxTokenKey)
		if !exist {
			tokenState = "No Bearer token available"
		} else {
			tokenState = tokenSaved.(token.Token).GetTokenState()
		}

		var redisKey string
		switch path := r.Request.URL.Path; {
		// TODO refactor condition
		case strings.HasSuffix(path, "/getposts") || strings.HasSuffix(path, "/posts") || strings.HasSuffix(path, "/post"):

			type Category struct {
				IsMemberOnly *bool `json:"isMemberOnly,omitempty"`
			}

			type ItemContent struct {
				APIData []interface{} `json:"apiData"`
			}
			type Item struct {
				Content    ItemContent `json:"content"`
				Categories []Category  `json:"categories"`
			}
			type Resp struct {
				Items []Item `json:"_items"`
			}

			var items Resp
			err = json.Unmarshal(body, &items)
			if err != nil {
				logger.Errorf("Unmarshal post encountered error: %v", err)
				return err
			}

			// truncate the content if the user is not a member and the post falls into a member only category
			if tokenState == token.OK {
				// TODO refactor redis cache code
				redisKey = fmt.Sprintf("%s.%s.%s.%s", "mm-apigateway", "post", "member", c.Request.RequestURI)
			} else {
				// TODO refactor redis cache code
				redisKey = fmt.Sprintf("%s.%s.%s.%s", "mm-apigateway", "post", "notmember", c.Request.RequestURI)

				// modify body if the item falls into a "member only" category
				for i, item := range items.Items {
					for _, category := range item.Categories {
						if category.IsMemberOnly != nil && *category.IsMemberOnly {
							truncatedAPIData := item.Content.APIData[0:3]
							body, err = sjson.SetBytes(body, fmt.Sprintf("_items.%d.content.apiData", i), truncatedAPIData)
							if err != nil {
								logger.Errorf("encounter error when truncating apiData:", err)
								return err
							}
							break
						}
					}
				}
			}

			// remove html because only apidata is useful and html contains full content
			for i, _ := range items.Items {
				body, err = sjson.DeleteBytes(body, fmt.Sprintf("_items.%d.content.html", i))
				if err != nil {
					logger.Errorf("encounter error when deleting html:", err)
					return err
				}
			}
			// TODO refactor redis cache code
			err = rdb.Set(context.TODO(), redisKey, body, time.Duration(cacheTTL)*time.Second).Err()
			if err != nil {
				logger.Warnf("setting redis cache(%s) encountered error: %v", redisKey, err)
			}
		default:
		}

		b, err := json.Marshal(Reply{
			TokenState:  tokenState,
			MemberState: MemberStateNone,
			Data:        json.RawMessage(body),
		})

		if err != nil {
			logger.Errorf("Marshalling reply encountered error: %v", err)
			return err
		}

		r.Body = io.NopCloser(bytes.NewReader(b))
		r.ContentLength = int64(len(b))
		r.Header.Set("Content-Length", strconv.Itoa(len(b)))
		return nil
	}
}

func NewSingleHostReverseProxy(target *url.URL, pathBaseToStrip string, rdb Rediser, cacheTTL int) func(c *gin.Context) {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		if strings.HasSuffix(pathBaseToStrip, "/") {
			pathBaseToStrip = pathBaseToStrip + "/"
		}
		req.URL.Path = strings.TrimPrefix(req.URL.Path, pathBaseToStrip)
		req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, pathBaseToStrip)

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path, req.URL.RawPath = joinURLPath(target, req.URL)

		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}

		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}
	return func(c *gin.Context) {
		// TODO refactor modification and cache code
		var tokenState string

		tokenSaved, exist := c.Get(middleware.GCtxTokenKey)
		if !exist {
			tokenState = "No Bearer token available"
		} else {
			tokenState = tokenSaved.(token.Token).GetTokenState()
		}

		switch path := c.Request.URL.Path; {
		case strings.HasSuffix(path, "/getposts") || strings.HasSuffix(path, "/posts") || strings.HasSuffix(path, "/post"):
			// Try to read cache first
			var key string
			if tokenState != token.OK {
				key = fmt.Sprintf("%s.%s.%s.%s", "mm-apigateway", "post", "notmember", c.Request.RequestURI)
			} else {
				key = fmt.Sprintf("%s.%s.%s.%s", "mm-apigateway", "post", "member", c.Request.RequestURI)
			}

			cmd := rdb.Get(context.TODO(), key)
			// cache doesn't exist, do fetch reverse proxy
			if cmd == nil {
				break
			}
			body, err := cmd.Bytes()
			// cache can't be understood, do fetch reverse proxy
			if err != nil {
				break
			}

			c.AbortWithStatusJSON(http.StatusOK, Reply{
				TokenState:  tokenState,
				MemberState: MemberStateNone,
				Data:        json.RawMessage(body),
			})
			return
		}

		reverseProxy := httputil.ReverseProxy{Director: director}
		reverseProxy.ModifyResponse = ModifyReverseProxyResponse(c, rdb, cacheTTL)
		reverseProxy.ServeHTTP(c.Writer, c.Request)
	}
}

type Reply struct {
	TokenState  interface{} `json:"tokenState"`
	MemberState interface{} `json:"memberState"`
	Data        interface{} `json:"data,omitempty"`
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
	v1tokenStateRouter := v1Router.Use(GetIDTokenOnly(server))
	v1tokenStateRouter.GET("/tokenState", func(c *gin.Context) {
		t := c.Value(middleware.GCtxTokenKey).(token.Token)
		if t == nil {
			c.JSON(http.StatusBadRequest, Reply{
				TokenState: nil,
			})
			return
		}
		c.JSON(http.StatusOK, Reply{
			TokenState:  t.GetTokenState(),
			MemberState: MemberStateNone,
		})
	})

	// Private API
	// v1 User
	// It will save FirebaseClient and FirebaseDBClient to *gin.context, and *gin.context to *context
	v1TokenAuthenticatedWithFirebaseRouter := v1Router.Use(AuthenticateIDToken(server), GinContextToContextMiddleware(server), FirebaseClientToContextMiddleware(server), FirebaseDBClientToContextMiddleware(server))
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
	v0tokenStateRouter := v0Router.Use(GetIDTokenOnly(server))
	proxyURL, err := url.Parse(server.Conf.V0RESTfulSrvTargetURL)
	if err != nil {
		return err
	}

	v0tokenStateRouter.Any("/*wildcard", NewSingleHostReverseProxy(proxyURL, v0Router.BasePath(), server.Rdb, server.Conf.RedisService.Cache.TTL))

	return nil
}
