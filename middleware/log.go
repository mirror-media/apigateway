package middleware

import (
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/mirror-media/apigateway/token"
	"github.com/sirupsen/logrus"
)

type LogrusMemberHook struct {
	firebaseID string
	email      string
	tokenState string
	isVerified bool
}

func (h LogrusMemberHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h LogrusMemberHook) Fire(e *logrus.Entry) error {
	data := map[string]interface{}{
		"firebaseId":      h.firebaseID,
		"email":           h.email,
		"isEmailVerified": h.isVerified,
		"tokenState":      h.tokenState,
	}
	m, ok := e.Data["logging.googleapis.com/labels"]
	if !ok {
		e.Data["logging.googleapis.com/labels"] = data
	} else {
		for k, v := range data {
			m.(map[string]interface{})[k] = v
		}
	}
	return nil
}

type FirebaseTokenClaims struct {
	jwt.RegisteredClaims
	Email           string `json:"email,omitempty"`
	IsEmailVerified bool   `json:"email_verified,omitempty"`
}

func AddFirebaseTokenInfoToLogrusHook(firebaseClient *auth.Client) func(c *gin.Context) {
	return func(c *gin.Context) {
		var firebaseID, email, tokenState string
		var isVerified bool

		v, exists := c.Get(GCtxTokenKey)
		var ok bool
		var t *token.FirebaseToken
		if exists {
			t, ok = v.(*token.FirebaseToken)
		}

		if exists && ok {
			tokenState = t.GetTokenState()
			claims := FirebaseTokenClaims{}
			ts, _ := t.GetTokenString()
			_, _, _ = new(jwt.Parser).ParseUnverified(ts, &claims)
			email = claims.Email
			isVerified = claims.IsEmailVerified
			firebaseID = claims.Subject
		}

		hook := LogrusMemberHook{
			firebaseID: firebaseID,
			email:      email,
			isVerified: isVerified,
			tokenState: tokenState,
		}

		logrus.AddHook(hook)

		c.Next()
	}
}
