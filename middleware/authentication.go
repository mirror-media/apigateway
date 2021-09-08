package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/db"
	"github.com/gin-gonic/gin"
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
			c.Next()
			return
		}
		c.Set(GCtxTokenKey, token)
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
			c.Next()
			return
		}

		if tt.GetTokenState() != token.OK {
			logger.Info(tt.GetTokenState())
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
		c.Next()
	}
}

func GinContextToContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), CtxGinContexKey, c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func FirebaseClientToContextMiddleware(firebaseClient *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), CtxFirebaseClientKey, firebaseClient)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func FirebaseDBClientToContextMiddleware(firebaseDatabaseClient *db.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), CtxFirebaseDatabaseClientKey, firebaseDatabaseClient)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
