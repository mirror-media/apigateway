package mutationgraph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	graphql99 "github.com/99designs/gqlgen/graphql"
	"github.com/machinebox/graphql"
	"github.com/mirror-media/apigateway/config"
	"github.com/mirror-media/apigateway/graph/member/model"
	"github.com/mirror-media/apigateway/middleware"
	"github.com/mirror-media/apigateway/payment"
	"github.com/sirupsen/logrus"

	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/db"
	"github.com/gin-gonic/gin"
)

type Connect struct {
	FirebaseID string `json:"firebaseId"`
}
type MemberConnect struct {
	Connect Connect `json:"connect"`
}

type Resolver struct {
	Client        *graphql.Client
	Conf          config.Conf
	UserSvrURL    string
	NewebpayStore payment.NewebPayStore
}

type WebhookPlayStoreResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    Data   `json:"data"`
}

type WebhookAppStoreResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Data struct {
	ID                      string `json:"id"`
	OrderNumber             string `json:"orderNumber"`
	Frequency               string `json:"frequency"`
	NextFrequency           string `json:"nextFrequency"`
	IsActive                bool   `json:"isActive"`
	GooglePlayStatus        string `json:"googlePlayStatus"`
	GooglePlayPurchaseToken string `json:"googlePlayPurchaseToken"`
	GooglePlayPackageName   string `json:"googlePlayPackageName"`
	PeriodFirstDatetime     string `json:"periodFirstDatetime"`
	PeriodEndDatetime       string `json:"periodEndDatetime"`
	PeriodCreateDatetime    string `json:"periodCreateDatetime"`
}

func (r Resolver) RetrieveExistingSubscriptionFromRemote(ctx context.Context, subscriptionID string) (firebaseID, frequency string, err error) {
	req := graphql.NewRequest("query ($id: ID!) { subscription(where: {id: $id}) { frequency, member { firebaseId } } }")
	req.Var("id", subscriptionID)

	var resp struct {
		Subscription *model.Subscription `json:"subscription"`
	}

	err = r.Client.Run(ctx, req, &resp)
	if err != nil {
		logrus.WithField("query", "RetrieveMemberFirebaseIDOfSubscriptionFromRemote").Error(err)
		return "", "", err
	} else if resp.Subscription == nil {
		return "", "", fmt.Errorf("subscription(%s) is not found", subscriptionID)
	} else if resp.Subscription.Member == nil {
		return "", "", fmt.Errorf("member of subscription(%s) is not found", subscriptionID)
	}
	return *resp.Subscription.Member.FirebaseID, resp.Subscription.Frequency.String(), err
}

func (r Resolver) RetrieveMerchandise(ctx context.Context, code string) (price float64, currency model.MerchandiseCurrencyType, state model.MerchandiseStateType, comment, description string, err error) {
	gql := `query ($code: String) {
  merchandise(where: {code: $code}) {
    price
    currency
    state
		comment
		desc
  }
}`
	req := graphql.NewRequest(gql)
	req.Var("code", code)

	var resp struct {
		Merchandise *model.Merchandise `json:"merchandise"`
	}

	err = r.Client.Run(ctx, req, &resp)
	if err != nil {
		logrus.WithField("query", "RetrieveMerchandise").Error(err)
		return 0, "", "", "", "", err
	} else if resp.Merchandise == nil {
		return 0, "", "", "", "", fmt.Errorf("merchandise with code %s is not found", code)
	}

	return *resp.Merchandise.Price, *resp.Merchandise.Currency, *resp.Merchandise.State, *resp.Merchandise.Comment, *resp.Merchandise.Desc, err
}

func (r Resolver) GetMemberIDFromRemote(ctx context.Context, firebaseID string) (string, error) {
	req := graphql.NewRequest(
		"query ($firebaseId: String) { member(where: {firebaseId: $firebaseId}) { id } }")
	req.Var("firebaseId", firebaseID)

	var resp struct {
		Member *model.Member `json:"member"`
	}

	err := r.Client.Run(ctx, req, &resp)
	if err != nil {
		logrus.WithField("query", "GetMemberIDFromRemote").Error(err)
		return "", err
	} else if resp.Member == nil {
		return "", fmt.Errorf("%s is not found", firebaseID)
	}
	return resp.Member.ID, err
}

func (r Resolver) GetFirebaseID(ctx context.Context) (string, error) {

	gCTX, err := GinContextFromContext(ctx)
	if err != nil {
		return "", err
	}

	id := gCTX.GetString(middleware.GCtxUserIDKey)

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

func contain(ss []string, s string) bool {
	for _, v := range ss {
		if s == v {
			return true
		}
	}
	return false
}

func createOrderNumberByTaipeiTZ(t time.Time, id uint64) (orderNumber string) {
	tz, _ := time.LoadLocation("Asia/Taipei")
	t = t.In(tz)
	prefix := "M"

	date := strconv.FormatInt(int64(t.Year()), 10)[2:] + fmt.Sprintf("%02d", t.Month()) + fmt.Sprintf("%02d", t.Day())

	orderNumber = fmt.Sprintf("%s%s%05d", prefix, date, id%10000)
	return orderNumber
}
