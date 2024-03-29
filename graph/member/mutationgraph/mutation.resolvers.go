package mutationgraph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	graphqlclient "github.com/machinebox/graphql"
	"github.com/mirror-media/apigateway/graph/member/model"
	"github.com/mirror-media/apigateway/graph/member/mutationgraph/generated"
	"github.com/mirror-media/apigateway/payment"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

func (r *mutationResolver) Createmember(ctx context.Context, data map[string]interface{}) (*model.MemberInfo, error) {
	if data == nil {
		return nil, fmt.Errorf("data cannot be null")
	}

	firebaseID, err := r.GetFirebaseID(ctx)
	if err != nil {
		return nil, err
	}
	data["firebaseId"] = firebaseID

	data["type"] = model.MemberTypeTypeNone
	data["dateJoined"] = time.Now().Format(time.RFC3339)

	// Construct GraphQL mutation

	preGQL := []string{"mutation($input: memberCreateInput!) {", "createmember(data: $input) {"}

	fieldsOnly := Map(GetPreloads(ctx), func(s string) string {
		ns := strings.Split(s, ".")
		return ns[len(ns)-1]
	})

	preGQL = append(preGQL, fieldsOnly...)
	preGQL = append(preGQL, "}", "}")
	gql := strings.Join(preGQL, "\n")

	req := graphqlclient.NewRequest(gql)
	req.Var("input", data)

	resp := struct {
		MemberInfo *model.MemberInfo `json:"createmember"`
	}{}

	err = r.Client.Run(ctx, req, &resp)

	if err != nil {
		logrus.WithField("mutation", "createmember")
		return nil, err
	}

	return resp.MemberInfo, err
}

func (r *mutationResolver) Updatemember(ctx context.Context, id string, data map[string]interface{}) (*model.MemberInfo, error) {
	firebaseID, err := r.GetFirebaseID(ctx)
	if err != nil {
		return nil, err
	}
	_id, err := r.GetMemberIDFromRemote(ctx, firebaseID)
	if err != nil {
		return nil, err
	} else if _id != id {
		return nil, fmt.Errorf("the id of firebaseId(%s) doesn't match id(%s)", firebaseID, id)
	}

	// Construct GraphQL mutation

	preGQL := []string{"mutation ($id: ID!, $input: memberUpdateInput) {", "updatemember(id: $id, data: $input) {"}

	fieldsOnly := Map(GetPreloads(ctx), func(s string) string {
		ns := strings.Split(s, ".")
		return ns[len(ns)-1]
	})

	preGQL = append(preGQL, fieldsOnly...)
	preGQL = append(preGQL, "}", "}")
	gql := strings.Join(preGQL, "\n")
	req := graphqlclient.NewRequest(gql)
	req.Var("id", id)
	req.Var("input", data)

	var resp struct {
		MemberInfo *model.MemberInfo `json:"updatemember"`
	}

	err = r.Client.Run(ctx, req, &resp)
	if err != nil {
		logrus.WithField("mutation", "updatemember").Error(err)
		return nil, err
	}

	return resp.MemberInfo, err
}

func (r *mutationResolver) UpsertAppSubscription(ctx context.Context, info model.SubscriptionAppUpsertInfo) (*model.SubscriptionUpsert, error) {
	logrus.Debug("UpsertAppSubscription for")
	firebaseID, err := r.GetFirebaseID(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Debug("UpsertAppSubscription for", firebaseID)
	var ret *model.SubscriptionUpsert
	switch info.Source {
	case model.UpsertSubscriptionAppSourceTypeAppStore:
		logrus.Debug("UpsertAppSubscription: AppStore")
		body, _ := json.Marshal(map[string]interface{}{
			"receiptData": info.VerificationData,
			"firebaseId":  firebaseID,
		})
		postBody := bytes.NewBuffer(body)
		resp, err := netClient.Post(r.Conf.ServiceEndpoints.AppStoreUpsertSubscription, "application/json", postBody)
		if err != nil {
			logrus.Error("posting request to AppStoreUpsertSubscription,"+r.Conf.ServiceEndpoints.AppStoreUpsertSubscription+" ,failed:", err)
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logrus.Error("posting request to AppStoreUpsertSubscription,"+r.Conf.ServiceEndpoints.AppStoreUpsertSubscription+" ,failed:", resp.StatusCode)
			return nil, fmt.Errorf("internal error")
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			logrus.Error("parsing play store subscription webhook body encountered an error:", err)
			return nil, fmt.Errorf("internal error")
		}

		var respData WebhookAppStoreResponse
		err = json.Unmarshal(body, &respData)
		if err != nil {
			logrus.Error("unmarshalling play store subscription webhook body encountered an error:", err, string(body))
			return nil, fmt.Errorf("internal error")
		}

		if respData.Status == "success" {
			ret = &model.SubscriptionUpsert{
				Success: true,
			}
		} else {
			ret = &model.SubscriptionUpsert{}
			err = fmt.Errorf(respData.Message)
		}
		return ret, err
	case model.UpsertSubscriptionAppSourceTypeGooglePlay:
		logrus.Debug("UpsertAppSubscription: GooglePlay")
		body, _ := json.Marshal(map[string]interface{}{
			"purchaseToken":  info.VerificationData,
			"subscriptionId": info.ProductID,
			"firebaseId":     firebaseID,
			"packageName":    info.PackageName,
		})
		postBody := bytes.NewBuffer(body)
		resp, err := netClient.Post(r.Conf.ServiceEndpoints.PlayStoreUpsertSubscription, "application/json", postBody)
		if err != nil {
			logrus.Error("posting request to PlayStoreUpsertSubscription,"+r.Conf.ServiceEndpoints.PlayStoreUpsertSubscription+" ,failed:", err)
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logrus.Error("posting request to PlayStoreUpsertSubscription,"+r.Conf.ServiceEndpoints.PlayStoreUpsertSubscription+" ,failed:", resp.StatusCode)
			return nil, fmt.Errorf("internal error")
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			logrus.Error("parsing play store subscription webhook body encountered an error:", err)
			return nil, fmt.Errorf("internal error")
		}

		var respData WebhookPlayStoreResponse
		err = json.Unmarshal(body, &respData)
		if err != nil {
			logrus.Error("unmarshalling play store subscription webhook body encountered an error:", err, string(body))
			return nil, fmt.Errorf("internal error")
		}

		if respData.Status == "success" {
			ret = &model.SubscriptionUpsert{
				Success: true,
			}
		} else {
			ret = &model.SubscriptionUpsert{}
			err = fmt.Errorf(respData.Message)
		}
		return ret, err

	default:
		return nil, fmt.Errorf("unknown source: %s", info.Source)
	}
	return ret, err
}

func (r *mutationResolver) CreateSubscriptionRecurring(ctx context.Context, data map[string]interface{}, info model.SubscriptionRecurringCreateInfo) (*model.SubscriptionCreation, error) {
	firebaseID, err := r.GetFirebaseID(ctx)
	if err != nil {
		return nil, err
	}

	data["member"] = MemberConnect{
		Connect: Connect{
			FirebaseID: firebaseID,
		},
	}

	frequency, ok := data["frequency"].(string)
	if !ok {
		return nil, fmt.Errorf("%v cannot be converted to string", data["frequency"])
	}
	price, currency, state, comment, description, err := r.RetrieveMerchandise(ctx, frequency)
	if err != nil {
		return nil, err
	}
	if state != model.MerchandiseStateTypeActive {
		return nil, fmt.Errorf("frequency(%s) is not %s", data["frequency"], model.MerchandiseStateTypeActive)
	}
	data["nextFrequency"] = data["frequency"]
	data["amount"] = price
	data["currency"] = currency
	data["comment"] = comment
	data["desc"] = description
	data["orderNumber"] = "preparing-order-" + xid.New().String()

	// Construct GraphQL mutation

	preGQL := []string{"mutation ($input: subscriptionCreateInput) {", "createsubscription(data: $input) {"}

	subscriptionFieldsOnly := Map(GetPreloads(ctx), func(s string) string {
		ns := strings.Split(s, ".")
		if ns[0] == "subscription" && len(ns) == 2 {
			return ns[len(ns)-1]
		} else {
			return ""
		}
	})

	preGQL = append(preGQL, subscriptionFieldsOnly...)
	if !contain(subscriptionFieldsOnly, "createdAt") {
		preGQL = append(preGQL, "createdAt")
	}
	if !contain(subscriptionFieldsOnly, "id") {
		preGQL = append(preGQL, "id")
	}
	preGQL = append(preGQL, "}", "}")
	gql := strings.Join(preGQL, "\n")
	req := graphqlclient.NewRequest(gql)
	req.Var("input", data)

	var resp struct {
		SubscriptionInfo *model.SubscriptionInfo `json:"createsubscription"`
	}

	err = r.Client.Run(ctx, req, &resp)

	if err != nil {
		logrus.WithField("mutation", "createsubscription").Error(err)
		return nil, err
	}

	id, _ := strconv.ParseUint(resp.SubscriptionInfo.ID, 10, 64)
	orderNumber := createOrderNumberByTaipeiTZ(time.Now(), id)

	resp.SubscriptionInfo.OrderNumber = &orderNumber

	gql = `
mutation ($id: ID!, $orderNumber: String!) {
  updatesubscription(id: $id, data: {orderNumber: $orderNumber}) {
    orderNumber
  }
}
`

	req = graphqlclient.NewRequest(gql)
	req.Var("id", resp.SubscriptionInfo.ID)
	req.Var("orderNumber", orderNumber)

	err = r.Client.Run(ctx, req, nil)
	if err != nil {
		err = errors.Wrapf(err, "update odernumber to subscription(%s) encounter error", resp.SubscriptionInfo.ID)
		logrus.WithField("mutation", "createsubscription.updatesubscription").Error(err)
		return nil, err
	}

	createAt := *resp.SubscriptionInfo.CreatedAt

	t, err := time.Parse(time.RFC3339, createAt)
	if err != nil {
		return nil, err
	}

	creationTimeUnix := t.Unix()

	payload, err := r.NewebpayStore.CreateNewebpayAgreementPayload(payment.NewebpayAgreementInfo{
		Amount:              int(price),
		Email:               data["email"].(string),
		IsAbleToModifyEmail: r.NewebpayStore.IsAbleToModifyEmail,
		LoginType:           r.NewebpayStore.LoginType,
		RespondType:         r.NewebpayStore.RespondType,
		ItemDesc:            description,
		OrderComment:        comment,
		TokenTerm:           firebaseID,
	}, payment.PurchaseInfo{
		Merchandise: payment.Merchandise{
			Code:   frequency,
			Amount: price,
		},
		PurchasedAtUnixTime: creationTimeUnix,
		OrderNumber:         orderNumber,
		MemberFirebaseID:    firebaseID,
		ReturnPath:          info.ReturnToPath,
	})
	if err != nil {
		return nil, err
	}

	return &model.SubscriptionCreation{
		Subscription:    resp.SubscriptionInfo,
		NewebpayPayload: &payload,
	}, err
}

func (r *mutationResolver) CreatesSubscriptionOneTime(ctx context.Context, data map[string]interface{}, info model.SubscriptionOneTimeCreateInfo) (*model.SubscriptionCreation, error) {
	if data == nil {
		return nil, fmt.Errorf("data cannot be null")
	}

	firebaseID, err := r.GetFirebaseID(ctx)
	if err != nil {
		return nil, err
	}

	data["member"] = MemberConnect{
		Connect: Connect{
			FirebaseID: firebaseID,
		},
	}
	data["frequency"] = model.SubscriptionFrequencyTypeOneTime.String()
	data["nextFrequency"] = model.SubscriptionNextFrequencyTypeNone.String()

	price, currency, state, comment, description, err := r.RetrieveMerchandise(ctx, model.SubscriptionFrequencyTypeOneTime.String())
	if err != nil {
		return nil, err
	}
	if state != model.MerchandiseStateTypeActive {
		return nil, fmt.Errorf("frequency(%s) is not %s", model.SubscriptionFrequencyTypeOneTime, model.MerchandiseStateTypeActive)
	}
	data["amount"] = price
	data["currency"] = currency
	data["comment"] = comment
	data["desc"] = description
	data["orderNumber"] = "preparing-order-" + xid.New().String()

	// Construct GraphQL mutation

	preGQL := []string{"mutation ($input: subscriptionCreateInput) {", "createsubscription(data: $input) {"}

	subscriptionFieldsOnly := Map(GetPreloads(ctx), func(s string) string {
		ns := strings.Split(s, ".")
		if ns[0] == "subscription" && len(ns) == 2 {
			return ns[len(ns)-1]
		} else {
			return ""
		}
	})

	preGQL = append(preGQL, subscriptionFieldsOnly...)
	if !contain(subscriptionFieldsOnly, "createdAt") {
		preGQL = append(preGQL, "createdAt")
	}
	if !contain(subscriptionFieldsOnly, "id") {
		preGQL = append(preGQL, "id")
	}
	preGQL = append(preGQL, "}", "}")
	gql := strings.Join(preGQL, "\n")
	req := graphqlclient.NewRequest(gql)
	req.Var("input", data)

	var resp struct {
		SubscriptionInfo *model.SubscriptionInfo `json:"createsubscription"`
	}

	err = r.Client.Run(ctx, req, &resp)

	if err != nil {
		logrus.WithField("mutation", "createsubscription").Error(err)
		return nil, err
	}
	id, _ := strconv.ParseUint(resp.SubscriptionInfo.ID, 10, 64)
	orderNumber := createOrderNumberByTaipeiTZ(time.Now(), id)

	resp.SubscriptionInfo.OrderNumber = &orderNumber

	gql = `
	mutation ($id: ID!, $orderNumber: String!) {
	  updatesubscription(id: $id, data: {orderNumber: $orderNumber}) {
	    orderNumber
	  }
	}
`

	req = graphqlclient.NewRequest(gql)
	req.Var("id", resp.SubscriptionInfo.ID)
	req.Var("orderNumber", orderNumber)

	err = r.Client.Run(ctx, req, nil)
	if err != nil {
		err = errors.Wrapf(err, "update odernumber to subscription(%s) encounter error", resp.SubscriptionInfo.ID)
		logrus.WithField("mutation", "createsubscription.updatesubscription").Error(err)
		return nil, err
	}

	createAt := *resp.SubscriptionInfo.CreatedAt

	t, err := time.Parse(time.RFC3339, createAt)
	if err != nil {
		return nil, err
	}

	creationTimeUnix := t.Unix()

	payload, err := r.NewebpayStore.CreateNewebpayMPGPayload(payment.NewebpayMGPInfo{
		Amount:              int(price),
		Email:               data["email"].(string),
		IsAbleToModifyEmail: r.NewebpayStore.IsAbleToModifyEmail,
		LoginType:           r.NewebpayStore.LoginType,
		RespondType:         r.NewebpayStore.RespondType,
		ItemDescription:     description,
		OrderComment:        orderNumber,
		TokenTerm:           firebaseID,
	}, payment.PurchaseInfo{
		Merchandise: payment.Merchandise{
			Code:      model.SubscriptionFrequencyTypeOneTime.String(),
			PostID:    data["postId"].(string),
			PostSlug:  info.PostSlug,
			PostTitle: info.PostTitle,
			Amount:    price,
		},
		PurchasedAtUnixTime: creationTimeUnix,
		OrderNumber:         orderNumber,
		MemberFirebaseID:    firebaseID,
		ReturnPath:          info.ReturnToPath,
	})
	if err != nil {
		return nil, err
	}

	return &model.SubscriptionCreation{
		Subscription:    resp.SubscriptionInfo,
		NewebpayPayload: &payload,
	}, err
}

func (r *mutationResolver) Updatesubscription(ctx context.Context, id string, data map[string]interface{}) (*model.SubscriptionInfo, error) {
	if data == nil {
		return nil, fmt.Errorf("data cannot be null")
	}

	firebaseID, err := r.GetFirebaseID(ctx)
	if err != nil {
		return nil, err
	}

	if isCanceled, ok := data["isCanceled"]; ok && (isCanceled != true) {
		return nil, errors.New("invalid mutation: you may not un-cancel the subscription")
	}

	_firebaseID, _frequency, err := r.RetrieveExistingSubscriptionFromRemote(ctx, id)
	if err != nil {
		return nil, err
	} else if _firebaseID != firebaseID {
		return nil, fmt.Errorf("you do not have access to this resource, subscription(%s)", id)
	} else if _frequency == model.SubscriptionFrequencyTypeOneTime.String() {
		return nil, fmt.Errorf("%s subscription cannot be updated", _frequency)
	}

	if nextFrequency, ok := data["nextFrequency"]; ok {
		nextFrequencyString, _ := nextFrequency.(string)
		_, _, state, _, _, err := r.RetrieveMerchandise(ctx, nextFrequencyString)
		if err != nil {
			return nil, err
		}
		if state != model.MerchandiseStateTypeActive {
			return nil, fmt.Errorf("frequency(%s) is not %s", model.SubscriptionFrequencyTypeOneTime, model.MerchandiseStateTypeActive)
		}
	}

	// Construct GraphQL mutation

	preGQL := []string{"mutation ($id: ID!, $input: subscriptionUpdateInput) {", "updatesubscription(id: $id, data: $input) {"}

	fieldsOnly := Map(GetPreloads(ctx), func(s string) string {
		ns := strings.Split(s, ".")
		return ns[len(ns)-1]
	})

	preGQL = append(preGQL, fieldsOnly...)
	preGQL = append(preGQL, "}", "}")
	gql := strings.Join(preGQL, "\n")
	req := graphqlclient.NewRequest(gql)
	req.Var("id", id)
	req.Var("input", data)

	var resp struct {
		SubscriptionInfo *model.SubscriptionInfo `json:"updatesubscription"`
	}

	err = r.Client.Run(ctx, req, &resp)

	if err != nil {
		logrus.WithField("mutation", "updatesubscription").Error(err)
		return nil, err
	}

	return resp.SubscriptionInfo, err
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

type mutationResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//  - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//    it when you're done.
//  - You have helper methods in this file. Move them out to keep these resolver files clean.
var netClient = &http.Client{
	Timeout: time.Second * 30,
}
