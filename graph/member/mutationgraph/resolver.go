package mutationgraph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"context"
	"errors"
	"fmt"

	graphql99 "github.com/99designs/gqlgen/graphql"
	"github.com/machinebox/graphql"
	"github.com/mirror-media/apigateway/config"
	"github.com/mirror-media/apigateway/graph/member/model"
	"github.com/mirror-media/apigateway/middleware"
	"github.com/sirupsen/logrus"

	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/db"
	"github.com/gin-gonic/gin"
)

type Resolver struct {
	// Token      token.Token
	Client     *graphql.Client
	Conf       config.Conf
	UserSvrURL string
}

func (r Resolver) RetrieveExistingSubscriptionFromRemote(ctx context.Context, subscriptionID string) (firebaseID, frequency string, err error) {
	req := graphql.NewRequest("query ($id: ID!) { subscription(where: {id: $id}) { frequency, member { firebaseId } } }")
	req.Var("id", subscriptionID)

	var resp struct {
		Data *struct {
			Subscription *model.Subscription `json:"subscription"`
		} `json:"data"`
	}

	err = r.Client.Run(ctx, req, &resp)
	checkAndPrintGraphQLError(logrus.WithField("query", "RetrieveMemberFirebaseIDOfSubscriptionFromRemote"), err)
	if err != nil {
		return "", "", err
	} else if resp.Data.Subscription.Member == nil {
		return "", "", fmt.Errorf("member of subscription(%s) is not found", subscriptionID)
	}
	return *resp.Data.Subscription.Member.FirebaseID, resp.Data.Subscription.Frequency.String(), err
}

func (r Resolver) RetrieveMerchandise(ctx context.Context, code string) (price float64, currency string, state model.MerchandiseStateType, err error) {
	gql := `query ($code: String) {
  merchandise(where: {code: $code}) {
    price
    currency
    state
  }
}`
	req := graphql.NewRequest(gql)
	req.Var("code", code)

	var resp struct {
		Data *struct {
			Merchandise *model.Merchandise `json:"merchandise"`
		} `json:"data"`
	}

	err = r.Client.Run(ctx, req, &resp)
	checkAndPrintGraphQLError(logrus.WithField("query", "GetIDFromRemote"), err)
	if err != nil {
		return 0, "", "", err
	} else if resp.Data.Merchandise == nil {
		return 0, "", "", fmt.Errorf("merchandise with code %s is not found", code)
	}
	return *resp.Data.Merchandise.Price, *resp.Data.Merchandise.Code, *resp.Data.Merchandise.State, err
}

func (r Resolver) GetMemberIDFromRemote(ctx context.Context, firebaseID string) (string, error) {
	req := graphql.NewRequest(
		"query ($firebaseId: String) { member(where: {firebaseId: $firebaseId}) { id } }")
	req.Var("firebaseId", firebaseID)

	var resp struct {
		Data *struct {
			Member *model.Member `json:"member"`
		} `json:"data"`
	}

	err := r.Client.Run(ctx, req, &resp)
	checkAndPrintGraphQLError(logrus.WithField("query", "GetIDFromRemote"), err)
	if err != nil {
		return "", err
	} else if resp.Data.Member == nil {
		return "", fmt.Errorf("%s is not found", firebaseID)
	}
	return resp.Data.Member.ID, err
}

func (r Resolver) GetFirebaseID(ctx context.Context) (string, error) {

	gCTX, err := GinContextFromContext(ctx)
	if err != nil {
		return "", err
	}

	id, ok := gCTX.Value(middleware.GCtxUserIDKey).(string)
	if !ok {
		err = fmt.Errorf("fail to get firebaseID")
	}

	return id, err
}

func (r Resolver) IsRequestMatchingRequesterFirebaseID(ctx context.Context, userID string) (bool, error) {

	gCTX, err := GinContextFromContext(ctx)
	if err != nil {
		return false, err
	}

	if userID != gCTX.Value(middleware.GCtxUserIDKey).(string) {
		return false, fmt.Errorf("member id(%s) is not allowed to perfrom action against member id(%s)", gCTX.Value(middleware.GCtxUserIDKey), userID)
	}
	return true, nil
}

func GinContextFromContext(ctx context.Context) (*gin.Context, error) {
	ginContext := ctx.Value(middleware.CtxGinContexKey)
	if ginContext == nil {
		err := fmt.Errorf("could not retrieve gin.Context")
		logrus.Error(err)
		return nil, err
	}

	gc, ok := ginContext.(*gin.Context)
	if !ok {
		err := fmt.Errorf("gin.Context has wrong type")
		logrus.Error(err)
		return nil, err
	}
	return gc, nil
}

func FirebaseClientFromContext(ctx context.Context) (*auth.Client, error) {
	gCTX, err := GinContextFromContext(ctx)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	logger := logrus.WithFields(logrus.Fields{
		"path": gCTX.FullPath(),
	})
	firebaseClientCtx := ctx.Value(middleware.CtxFirebaseClientKey)

	client, ok := firebaseClientCtx.(*auth.Client)
	if !ok {
		err := fmt.Errorf("auth.Client has wrong type")
		logger.Error(err)
		return nil, err
	}
	return client, nil
}

func FirebaseDatabaseClientFromContext(ctx context.Context) (*db.Client, error) {
	gCTX, err := GinContextFromContext(ctx)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	logger := logrus.WithFields(logrus.Fields{
		"path": gCTX.FullPath(),
	})
	firebaseDatabaseClientCtx := ctx.Value(middleware.CtxFirebaseDatabaseClientKey)

	client, ok := firebaseDatabaseClientCtx.(*db.Client)
	if !ok {
		err := errors.New("db.Client has wrong type")
		logger.Error(err)
		return nil, err
	}
	return client, nil
}

func GetPreloads(ctx context.Context) []string {
	return GetNestedPreloads(
		graphql99.GetOperationContext(ctx),
		graphql99.CollectFieldsCtx(ctx, nil),
		"",
	)
}

func GetNestedPreloads(ctx *graphql99.OperationContext, fields []graphql99.CollectedField, prefix string) (preloads []string) {
	for _, column := range fields {
		prefixColumn := GetPreloadString(prefix, column.Name)
		preloads = append(preloads, prefixColumn)
		preloads = append(preloads, GetNestedPreloads(ctx, graphql99.CollectFields(ctx, column.Selections, nil), prefixColumn)...)
	}
	return
}

func GetPreloadString(prefix, name string) string {
	if len(prefix) > 0 {
		return prefix + "." + name
	}
	return name
}

func Map(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

func checkAndPrintGraphQLError(logger *logrus.Entry, err error) {
	if err != nil {
		logger.Infof("GraphQL request received error from:%v", err)
	}
}
