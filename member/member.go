// Package member defines the member related functions
package member

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/machinebox/graphql"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"cloud.google.com/go/pubsub"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/db"
	"github.com/mirror-media/apigateway/config"
	"github.com/mirror-media/apigateway/token"
	"github.com/pkg/errors"
)

const (
	MsgAttrKeyAction     = "action"
	MsgAttrKeyFirebaseID = "firebaseID"
)
const (
	MsgAttrValueDelete = "delete"
)

type Clients struct {
	sync.Once
	conf          config.Conf
	graphqlClient *graphql.Client
}

func (c *Clients) getGraphQLClient(userSvrToken string, serverConf config.Conf) (graphqlClient *graphql.Client, err error) {
	c.Do(func() {
		src := oauth2.StaticTokenSource(
			&oauth2.Token{
				AccessToken: userSvrToken,
				TokenType:   token.TypeJWT,
			},
		)
		httpClient := oauth2.NewClient(context.Background(), src)
		c.graphqlClient = graphql.NewClient(serverConf.ServiceEndpoints.UserGraphQL, graphql.WithHTTPClient(httpClient))
	})
	if c.graphqlClient == nil {
		return nil, errors.New("graphqlClient is nil")
	}
	return c.graphqlClient, nil
}

// singleton clients
var clients Clients

// DisableFirebaseUser disables the user in Firebase
func DisableFirebaseUser(parent context.Context, client *auth.Client, firebaseID string) (err error) {

	ctx, cancelDisable := context.WithCancel(parent)
	defer cancelDisable()
	params := (&auth.UserToUpdate{}).Disabled(true)
	_, err = client.UpdateUser(ctx, firebaseID, params)
	if err != nil {
		err = errors.WithMessagef(err, "fail to disable member(%s)", firebaseID)
		return err
	}
	return nil
}

// Delete performs a series of actions to revoke token, remove firebase user and request to disable the member in the DB
func Delete(parent context.Context, serverConf config.Conf, client *auth.Client, dbClient *db.Client, firebaseID string) (err error) {

	if err = revokeFirebaseToken(parent, client, dbClient, firebaseID); err != nil {
		return err
	} else if err = deleteFirebaseUser(parent, client, firebaseID); err != nil {
		return err
	}

	if err = publishDeleteMemberMessage(parent, serverConf.ProjectID, serverConf.PubSubTopicMember, firebaseID); err != nil {
		return err
	}

	return nil
}

func revokeFirebaseToken(parent context.Context, client *auth.Client, dbClient *db.Client, firebaseID string) (err error) {

	ctx, cancelRevoke := context.WithTimeout(parent, 10*time.Second)
	defer cancelRevoke()
	if err := client.RevokeRefreshTokens(ctx, firebaseID); err != nil {
		logrus.Errorf("error revoking tokens for user: %v, %v", firebaseID, err)
		return err
	}
	logrus.Infof("revoked tokens for user: %v", firebaseID)
	// accessing the user's TokenValidAfter
	ctx, cancelGetUser := context.WithTimeout(parent, 10*time.Second)
	defer cancelGetUser()
	u, err := client.GetUser(ctx, firebaseID)
	if err != nil {
		logrus.Errorf("error getting user %s: %v", firebaseID, err)
		return err
	}
	timestamp := u.TokensValidAfterMillis / 1000
	logrus.Printf("the refresh tokens were revoked at: %d (UTC seconds) ", timestamp)
	// save revoked time metadata for the user
	ctx, cancelSetMetadataRevokeTime := context.WithTimeout(parent, 10*time.Second)
	defer cancelSetMetadataRevokeTime()
	if err := dbClient.NewRef("metadata/"+u.UID).Set(ctx, map[string]int64{"revokeTime": timestamp}); err != nil {
		logrus.Error(err)
		return err
	}

	return err
}

func deleteFirebaseUser(parent context.Context, client *auth.Client, firebaseID string) error {

	ctx, cancelDelete := context.WithCancel(parent)
	defer cancelDelete()
	err := client.DeleteUser(ctx, firebaseID)
	if err != nil {
		err = errors.WithMessagef(err, "member(%s) deletion failed", firebaseID)
		return err
	}
	return nil
}

func publishDeleteMemberMessage(parent context.Context, projectID string, topic string, firebaseID string) error {

	clientCTX, cancel := context.WithCancel(parent)
	defer cancel()
	client, err := pubsub.NewClient(clientCTX, projectID)
	if err != nil {
		err = errors.WithMessage(err, "error creating client for pubsub")
		return err
	}

	ctx, cancelPublish := context.WithCancel(clientCTX)
	defer cancelPublish()
	t := client.Topic(topic)
	result := t.Publish(ctx, &pubsub.Message{
		Attributes: map[string]string{
			MsgAttrKeyFirebaseID: firebaseID,
			MsgAttrKeyAction:     MsgAttrValueDelete,
		},
	})
	// Block until the result is returned and a server-generated
	// ID is returned for the published message.
	ctx, cancelGet := context.WithCancel(clientCTX)
	defer cancelGet()
	id, err := result.Get(ctx)
	if err != nil {
		errors.WithMessage(err, "get published message result has error")
		return err
	}
	logrus.Printf("Published member deletion message with custom attributes(firebaseID: %s); msg ID: %v", firebaseID, id)
	return nil
}

func SubscribeDeleteMember(parent context.Context, c config.Conf, userSvrToken token.Token) error {
	clientCTX, cancel := context.WithCancel(parent)
	defer cancel()
	client, err := pubsub.NewClient(clientCTX, c.ProjectID)
	if err != nil {
		return fmt.Errorf("pubsub.NewClient: %v", err)
	}
	defer client.Close()

	sub := client.Subscription(c.PubSubSubscribeMember)

	// Create a channel to handle messages to as they come in.
	cm := make(chan *pubsub.Message)
	defer close(cm)

	tokenString, err := userSvrToken.GetTokenString()
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
	graphqlClient := graphql.NewClient(c.ServiceEndpoints.UserGraphQL, graphql.WithHTTPClient(httpClient))

	// Handle individual messages in a goroutine.
	go func() {
		for msg := range cm {
			firebaseID := msg.Attributes[MsgAttrKeyFirebaseID]
			logrus.Infof("Got message to %s member: %s", msg.Attributes[MsgAttrKeyAction], firebaseID)

			switch msg.Attributes[MsgAttrKeyAction] {
			case MsgAttrValueDelete:
				if err := requestToDeleteMember(userSvrToken, graphqlClient, firebaseID); err == nil {
					msg.Ack()
				}
			default:
				logrus.Errorf("action(%s) is not supported", msg.Attributes[MsgAttrKeyAction])
			}
		}
	}()

	// Receive messages for 10 seconds.
	ctx, cancelReceive := context.WithTimeout(clientCTX, 10*time.Second)
	defer cancelReceive()
	// Receive blocks until the context is cancelled or an error occurs.
	logrus.Infof("Pulling subscription: %s", c.PubSubSubscribeMember)
	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		cm <- msg
	})
	if err != nil {
		err = errors.Wrap(err, "receive failed")
		logrus.Error(err)
		return err
	}

	return nil

}

func requestToDeleteMember(userSvrToken token.Token, graphqlClient *graphql.Client, firebaseID string) (err error) {
	logrus.Infof("Request Saleor-mirror to delete member: %s", firebaseID)

	preGQL := []string{"mutation($firebaseId: String!) {", "deleteMember(firebaseId: $firebaseId) {"}

	preGQL = append(preGQL, "success")
	preGQL = append(preGQL, "}", "}")
	gql := strings.Join(preGQL, "\n")

	req := graphql.NewRequest(gql)
	req.Var("firebaseId", firebaseID)

	// Ask User service to delete the member
	var resp struct {
		DeleteMember *model.DeleteMember `json:"deleteMember"`
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = graphqlClient.Run(ctx, req, &resp); err == nil {
		logrus.Infof("Successfully delete member(%s)", firebaseID)
	} else {
		logrus.Errorf("Fail to delete member(%s):%v", firebaseID, err)
	}
	return err
}
