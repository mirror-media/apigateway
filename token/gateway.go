package token

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/dgrijalva/jwt-go/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

type Gateway struct {
	sync.RWMutex
	state                  *string
	secretName             *string
	secretVersion          *string
	tokenString            string
	parser                 jwt.Parser
	renewTokenTopic        string
	updateTokenTopic       string
	renewVersionPubChannel chan string
	updateTokenSubChannel  chan string
	// client                 *pubsub.Client
}

func (g *Gateway) GetTokenString() (string, error) {
	g.RLock()
	defer g.RUnlock()
	return g.tokenString, nil
}

var uErr *jwt.UnverfiableTokenError
var expErr *jwt.TokenExpiredError
var nbfErr *jwt.TokenNotValidYetError

// ExecuteTokenStateUpdate check the JWT token, but it does not verify the token signature
func (g *Gateway) ExecuteTokenStateUpdate() error {
	g.Lock()
	defer g.Unlock()
	token, _, err := g.parser.ParseUnverified(g.tokenString, nil)
	var s string
	if token.Valid {
		s = OK
	} else if xerrors.As(err, &uErr) {
		s = "that's not even a token"
	} else if xerrors.As(err, &expErr) {
		s = ("token has expired")
	} else if xerrors.As(err, &nbfErr) {
		s = "token is not valid yet"
	} else {
		s = fmt.Sprintf("couldn't handle this token:%s", err.Error())
	}
	g.state = &s
	return nil

}

func (g *Gateway) GetTokenState() string {
	if g.state == nil {
		g.ExecuteTokenStateUpdate()
	}
	g.RLock()
	if g.state == nil {
		return ""
	}
	defer g.RUnlock()
	return *g.state
}

// TODO
// func (g *Gateway) pubsub() error {

// 	ctx := context.Background()

// 	updateTopic := g.client.Topic(g.updateTokenTopicID)

// 	sub, _ := g.client.CreateSubscription(ctx, "subName", pubsub.SubscriptionConfig{
// 		Topic:            updateTopic,
// 		AckDeadline:      10 * time.Second,
// 		ExpirationPolicy: 25 * time.Hour,
// 	})

// 	sub.Receive(ctx, func(c context.Context, msg *pubsub.Message) {
// 		// TODO update the token
// 	})

// 	for {
// 		select {
// 		case version := <-g.renewVersionPubChannel:
// 			topic := g.client.Topic(g.renewTokenTopicID)
// 			message := map[string]string{
// 				"version":     version,
// 				"secret_name": *g.secretName,
// 			}

// 			messageJSON, err := json.Marshal(message)
// 			if err != nil {
// 				return err
// 			}

// 			result := topic.Publish(ctx, &pubsub.Message{
// 				Data: messageJSON,
// 			})
// 			_, err = result.Get(ctx)
// 			if err != nil {
// 				return errors.New(fmt.Sprintf("Get: %v", err))
// 			}
// 		}
// 	}
// }

// func (g *Gateway) UpdateToken() error {

// 	// FIXME
// 	g.Lock()
// 	defer g.Unlock()
// 	g.renewVersionPubChannel <- *g.secretVersion
// 	return nil
// }

func NewGatewayToken(tokenSecretName string, projectID string) (*Gateway, error) {

	// Create the client.
	ctx := context.Background()
	c, err := secretmanager.NewClient(ctx)
	if err != nil {
		err = errors.Wrap(err, "failed to setup client")
		return nil, err
	}

	latestSecretVersion := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, tokenSecretName)

	req := &secretmanagerpb.GetSecretVersionRequest{
		Name: latestSecretVersion,
	}

	v, err := c.GetSecretVersion(ctx, req)
	if err != nil {
		err = errors.Wrapf(err, "failed to get latest secret version of %s: %v", tokenSecretName)
		return nil, err
	}

	getSecretReq := &secretmanagerpb.AccessSecretVersionRequest{
		Name: latestSecretVersion,
	}

	secret, err := c.AccessSecretVersion(ctx, getSecretReq)
	if err != nil {
		err = errors.Wrapf(err, "failed to get latest version of secret data of %s: %v", tokenSecretName)
		return nil, err
	}

	var tokenSecret struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
	}

	err = json.Unmarshal(secret.GetPayload().GetData(), &tokenSecret)
	if err != nil {
		err = errors.Wrapf(err, "cannot unmarshal secret data of %s: %v", tokenSecretName)
		return nil, err
	}

	versionFragments := strings.Split(v.Name, "/")
	version := versionFragments[len(versionFragments)-1]
	logrus.Infof("Using gateway token version:%s", version)

	g := Gateway{
		secretVersion: &version,
		tokenString:   tokenSecret.Token,
	}
	// g.pubsub()
	return &g, nil
}
