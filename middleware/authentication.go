package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/db"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jensneuse/graphql-go-tools/pkg/astparser"
	"github.com/jensneuse/graphql-go-tools/pkg/graphql"
	"github.com/mirror-media/apigateway/token"
	"github.com/sirupsen/logrus"
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

// GetIDTokenOnly is a middleware to construct the token.Token interface
func GetIDTokenOnly(firebaseClient *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := logrus.WithFields(logrus.Fields{
			"path": c.FullPath(),
		})
		// Create a Token Instance
		authHeader := c.GetHeader("Authorization")
		token, err := token.NewFirebaseToken(authHeader, firebaseClient)
		if err != nil {
			logger.Info(err)
			ginContextToContextMiddleware(c)
			c.Next()
			return
		}
		c.Set(GCtxTokenKey, token)
		ginContextToContextMiddleware(c)
		c.Next()
	}
}

// ! Do NOT use it for production
// GetFirebaseIDUnverified is a middleware to construct the token.Token interface
func GetFirebaseIDUnverified(firebaseClient *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := logrus.WithFields(logrus.Fields{
			"path": c.FullPath(),
		})
		// Create a Token Instance
		authHeader := c.GetHeader("Authorization")
		token, err := token.NewFirebaseToken(authHeader, firebaseClient)
		if err != nil {
			logger.Info(err)
			ginContextToContextMiddleware(c)
			c.Next()
			return
		}
		c.Set(GCtxTokenKey, token)

		s, _ := token.GetTokenString()
		claims := jwt.StandardClaims{}
		_, _, err = new(jwt.Parser).ParseUnverified(s, &claims)

		if err == nil {
			c.Set(GCtxUserIDKey, claims.Subject)
		} else {
			c.Set(GCtxUserIDKey, "")
		}

		ginContextToContextMiddleware(c)
		c.Next()
	}
}

// SetUserID is a middleware to authenticate the request and save the result to the context
func SetUserID(firebaseClient *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := logrus.WithFields(logrus.Fields{
			"path": c.FullPath(),
		})

		// Create a Token Instance
		t := c.Value(GCtxTokenKey)
		if t == nil {
			err := errors.New("no token provided")
			logger.Info(err)
			return
		}
		tt, ok := t.(token.Token)
		if !ok {
			logger.Info(GCtxTokenKey + " cannot be casted to token.Token")
			ginContextToContextMiddleware(c)
			c.Next()
			return
		}

		if tt.GetTokenState() != token.OK {
			logger.Info(tt.GetTokenState())
			ginContextToContextMiddleware(c)
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Because GetTokenState() already fetch the public key and cache it. Here VerifyIDToken() would only verify the signature.
		tokenString, _ := tt.GetTokenString()
		idToken, _ := firebaseClient.VerifyIDToken(ctx, tokenString)
		c.Set(GCtxUserIDKey, idToken.Subject)
		ginContextToContextMiddleware(c)
		c.Next()
	}
}

// AuthenticateIDToken is a middleware to authenticate the request and save the result to the context
func AuthenticateIDToken(firebaseClient *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := logrus.WithFields(logrus.Fields{
			"path": c.FullPath(),
		})
		// Create a Token Instance
		t := c.Value(GCtxTokenKey)
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
		tokenString, _ := tt.GetTokenString()
		idToken, err := firebaseClient.VerifyIDToken(ctx, tokenString)
		if err != nil {
			logger.Info(err.Error())
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorReply{
				Errors: []Error{{Message: err.Error()}},
			})
			return
		}
		c.Set(GCtxUserIDKey, idToken.Subject)
		ginContextToContextMiddleware(c)
		c.Next()
	}
}

func AuthenticateMemberQueryAndFirebaseIDInArguments(c *gin.Context) {
	logger := logrus.WithFields(logrus.Fields{
		"path": c.FullPath(),
	})

	graphqlRequest := graphql.Request{}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Warn(err)
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("cannot read http request body"))
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	// t, err := graphqlRequest.OperationType()
	// if err != nil {
	// 	e := fmt.Errorf("cannot get operation type")
	// 	c.AbortWithError(http.StatusBadRequest, e)
	// 	logger.Warn(errors.Wrap(err, e.Error()))
	// 	return
	// }

	// if t != graphql.OperationTypeQuery && t != graphql.OperationTypeUnknown {
	// 	c.Next()
	// 	return
	// }

	reader := bytes.NewReader(body)
	if err := graphql.UnmarshalRequest(reader, &graphqlRequest); err != nil {
		logger.Warn(err)
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("cannot unmarshal graphql request"))
		return
	}

	graphqlType, _ := graphqlRequest.OperationType()
	if graphqlType != graphql.OperationTypeQuery {
		c.Next()
		return
	}

	query, report := astparser.ParseGraphqlDocumentString(graphqlRequest.Query)
	if report.HasErrors() {
		logger.Warn(report.Error())
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("cannot parse graphql query document"))
		return
	}

	// TODO This is so ugly. I can make my tear bust by simply looking at it.
	var isMemberWithArgumentsInQuery bool
	queryString := string(query.Input.RawBytes)
	whereArguments := make([]string, 0)
	for _, f := range query.Fields {
		if (string)(query.Input.RawBytes)[f.Name.Start:f.Name.End] == "member" {
			if f.HasArguments {
				isMemberWithArgumentsInQuery = isMemberWithArgumentsInQuery || true
				s := queryString
				rightToTrim := s
				var i uint32
				for i = 1; i < f.Arguments.RPAREN.LineStart; i++ {
					rightToTrim = rightToTrim[strings.Index(rightToTrim, "\n")+1:]
				}

				rightToTrim = rightToTrim[int64(f.Arguments.RPAREN.CharStart-1):]

				s = strings.TrimSuffix(s, rightToTrim)
				for i := uint32(1); i < f.Arguments.LPAREN.LineStart; i++ {
					s = s[strings.Index(s, "\n")+1:]
				}

				s = s[f.Arguments.LPAREN.CharStart:]
				s = s[len("where"):]
				s = strings.TrimLeft(s, ": \n")
				whereArguments = append(whereArguments, s)
			}
		}
	}
	if !isMemberWithArgumentsInQuery {
		c.Next()
		return
	}

	authenticatedID, ok := c.Value(GCtxUserIDKey).(string)
	if !ok {
		err = fmt.Errorf("user id from context. interface conversion: interface is nil, not string")
		logger.Info(err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	const FirebaseIDKey = "firebaseId"
	for _, whereArgument := range whereArguments {
		if strings.HasPrefix(whereArgument, "{") && strings.HasSuffix(whereArgument, "}") {
			logger.Infof("whereArgument:%+v\n", whereArgument)
			whereArgument = strings.Trim(whereArgument[1:len(whereArgument)-1], " ")
			idx := strings.Index(whereArgument, FirebaseIDKey)

			if idx == -1 {
				continue
			}

			var firebaseIDValue string
			firebaseIDValue = strings.TrimLeft(whereArgument[idx+len(FirebaseIDKey):], " :")
			idx = strings.IndexAny(firebaseIDValue, ", ")
			if idx != -1 {
				firebaseIDValue = firebaseIDValue[:idx+1]
			}

			if firebaseIDValue[0] == '$' {
				varName := firebaseIDValue[1:]
				type Where map[string]interface{}
				w := Where{}
				err := json.Unmarshal(graphqlRequest.Variables, &w)
				if err != nil {
					logrus.Warn(err)
					c.AbortWithError(http.StatusBadRequest, fmt.Errorf("cannot unmarshal firebaseId value"))
					return
				}
				v, ok := w[varName]
				if ok {
					if abortWithInvalidFirebaseID(c, v.(string), authenticatedID) {
						return
					}
				} else {
					err = fmt.Errorf("there is no variable called %v", varName)
					logger.Info(err)
					c.AbortWithError(http.StatusBadRequest, err)
				}
			} else if abortWithInvalidFirebaseID(c, firebaseIDValue[1:len(firebaseIDValue)-1], authenticatedID) {
				// firebaseIDValue == "...."
				return
			}
		} else {
			err = fmt.Errorf("where argument is not an object")
			logger.Info(err)
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
	}

	c.Next()
}

func abortWithInvalidFirebaseID(c *gin.Context, id, authenticatedID string) bool {
	isValid := id == authenticatedID
	if !isValid {
		err := fmt.Errorf("queried member firebase id(%s) != token's firebaseID(%s)", id, authenticatedID)
		c.AbortWithError(http.StatusForbidden, err)
	}
	return !isValid
}

// func GinContextToContextMiddleware() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		ctx := context.WithValue(c.Request.Context(), CtxGinContexKey, c)
// 		c.Request = c.Request.WithContext(ctx)
// 		c.Next()
// 	}
// }

func ginContextToContextMiddleware(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), CtxGinContexKey, c)
	c.Request = c.Request.WithContext(ctx)
}

func FirebaseClientToContextMiddleware(firebaseClient *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), CtxFirebaseClientKey, firebaseClient)
		c.Request = c.Request.WithContext(ctx)
		ginContextToContextMiddleware(c)
		c.Next()
	}
}

func FirebaseDBClientToContextMiddleware(firebaseDatabaseClient *db.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), CtxFirebaseDatabaseClientKey, firebaseDatabaseClient)
		c.Request = c.Request.WithContext(ctx)
		ginContextToContextMiddleware(c)
		c.Next()
	}
}
