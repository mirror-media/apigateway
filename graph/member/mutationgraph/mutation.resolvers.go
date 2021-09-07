package mutationgraph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"strings"
	"time"

	graphqlclient "github.com/machinebox/graphql"
	"github.com/mirror-media/apigateway/graph/member/model"
	"github.com/mirror-media/apigateway/graph/member/mutationgraph/generated"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

func (r *mutationResolver) Createmember(ctx context.Context, data *model.MemberCreateInput) (*model.MemberInfo, error) {
	firebaseID, err := r.GetFirebaseID(ctx)
	if err != nil {
		return nil, err
	}

	input := &model.MemberPrivateCreateInput{
		FirebaseID: firebaseID,
	}

	if data == nil {
		input = nil
	} else {
		input.Address = data.Address
		input.Birthday = data.Birthday
		input.Email = data.Email
		input.Type = model.MemberTypeTypeNone
		input.DateJoined = time.Now().Format(ISO8601Layout)
		input.Tos = data.Tos
		input.FirstName = data.FirstName
		input.LastName = data.LastName
		input.Name = data.Name
		input.Gender = data.Gender
		input.Phone = data.Phone
		input.Birthday = data.Birthday
		input.Address = data.Address
		input.Nickname = data.Nickname
		input.ProfileImage = data.ProfileImage
		input.City = data.City
		input.Country = data.Country
		input.District = data.District
	}

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
	req.Var("input", input)

	resp := struct {
		MemberInfo *model.MemberInfo `json:"createmember"`
	}{}

	err = r.Client.Run(ctx, req, &resp)

	checkAndPrintGraphQLError(logrus.WithField("mutation", "createmember"), err)

	return resp.MemberInfo, err
}

func (r *mutationResolver) Updatemember(ctx context.Context, id string, data *model.MemberUpdateInput) (*model.MemberInfo, error) {
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

	var input *model.MemberPrivateUpdateInput
	if data != nil {
		input = &model.MemberPrivateUpdateInput{
			Email:        data.Email,
			Tos:          data.Tos,
			FirstName:    data.FirstName,
			LastName:     data.LastName,
			Name:         data.Name,
			Gender:       data.Gender,
			Phone:        data.Phone,
			Birthday:     data.Birthday,
			Address:      data.Address,
			Nickname:     data.Nickname,
			ProfileImage: data.ProfileImage,
			City:         data.City,
			Country:      data.Country,
			District:     data.District,
		}
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
	req.Var("input", input)

	var resp struct {
		MemberInfo *model.MemberInfo `json:"updatemember"`
	}

	err = r.Client.Run(ctx, req, &resp)

	checkAndPrintGraphQLError(logrus.WithField("mutation", "createmember"), err)

	return resp.MemberInfo, err
}

func (r *mutationResolver) CreateSubscriptionRecurring(ctx context.Context, data *model.SubscriptionRecurringCreateInput) (*model.SubscriptionCreation, error) {
	firebaseID, err := r.GetFirebaseID(ctx)
	if err != nil {
		return nil, err
	}

	type Input struct {
		model.SubscriptionPrivateNoMemberCreateInput
		Member struct {
			Connect model.MemberWhereUniqueInput `json:"connect"`
		} `json:"member"`
	}

	var input Input
	if data != nil {
		input = Input{
			SubscriptionPrivateNoMemberCreateInput: model.SubscriptionPrivateNoMemberCreateInput{
				PaymentMethod:   &data.PaymentMethod,
				ApplepayPayment: data.ApplepayPayment,
				Desc:            data.Desc,
				Email:           &data.Email,
				Frequency:       &data.Frequency,
				NextFrequency:   (*model.SubscriptionNextFrequencyType)(&data.Frequency),
				Note:            data.Note,
				PromoteID:       data.PromoteID,
			},
		}

		input.Member.Connect.FirebaseID = firebaseID

		status := (model.SubscriptionStatusType)(data.Status)
		input.Status = &status

		orderNumber := xid.New().String()
		input.OrderNumber = &orderNumber

		price, currency, state, err := r.RetrieveMerchandise(ctx, data.Frequency.String())
		if err != nil {
			return nil, err
		}
		if state != model.MerchandiseStateTypeActive {
			return nil, fmt.Errorf("frequency(%s) is not %s", data.Frequency, model.MerchandiseStateTypeActive)
		}

		amount := int(price)
		input.Amount = &amount

		input.Currency = (*model.SubscriptionCurrencyType)(&currency)
	}

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
	preGQL = append(preGQL, "}", "}")
	gql := strings.Join(preGQL, "\n")
	req := graphqlclient.NewRequest(gql)
	req.Var("input", input)

	var resp struct {
		SubscriptionInfo *model.SubscriptionInfo `json:"createsubscription"`
	}

	err = r.Client.Run(ctx, req, &resp)

	checkAndPrintGraphQLError(logrus.WithField("mutation", "createsubscription"), err)

	// TODO newebpay
	return &model.SubscriptionCreation{
		Subscription: resp.SubscriptionInfo,
	}, err
}

func (r *mutationResolver) CreatesSubscriptionOneTime(ctx context.Context, data *model.SubscriptionOneTimeCreateInput) (*model.SubscriptionCreation, error) {
	if data == nil {
		return nil, fmt.Errorf("data cannot be null")
	}

	type Input struct {
		model.SubscriptionPrivateNoMemberCreateInput
		Member struct {
			Connect model.MemberWhereUniqueInput `json:"connect"`
		} `json:"member"`
	}

	input := Input{
		SubscriptionPrivateNoMemberCreateInput: model.SubscriptionPrivateNoMemberCreateInput{
			PaymentMethod:   &data.PaymentMethod,
			ApplepayPayment: data.ApplepayPayment,
			Desc:            data.Desc,
			Email:           &data.Email,
			Note:            data.Note,
			PromoteID:       data.PromoteID,
			PostID:          &data.PostID,
		},
	}

	firebaseID, err := r.GetFirebaseID(ctx)
	if err != nil {
		return nil, err
	}
	input.Member.Connect.FirebaseID = firebaseID

	frequency := model.SubscriptionFrequencyTypeOneTime
	input.Frequency = &frequency
	none := model.SubscriptionNextFrequencyTypeNone
	input.NextFrequency = &none

	status := (model.SubscriptionStatusType)(data.Status)
	input.Status = &status

	orderNumber := xid.New().String()
	input.OrderNumber = &orderNumber

	price, currency, state, err := r.RetrieveMerchandise(ctx, model.SubscriptionFrequencyTypeOneTime.String())
	if err != nil {
		return nil, err
	}
	if state != model.MerchandiseStateTypeActive {
		return nil, fmt.Errorf("frequency(%s) is not %s", model.SubscriptionFrequencyTypeOneTime, model.MerchandiseStateTypeActive)
	}

	amount := int(price)
	input.Amount = &amount

	input.Currency = (*model.SubscriptionCurrencyType)(&currency)

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
	preGQL = append(preGQL, "}", "}")
	gql := strings.Join(preGQL, "\n")
	req := graphqlclient.NewRequest(gql)
	req.Var("input", input)

	var resp struct {
		SubscriptionInfo *model.SubscriptionInfo `json:"createsubscription"`
	}

	err = r.Client.Run(ctx, req, &resp)

	checkAndPrintGraphQLError(logrus.WithField("mutation", "createsubscription"), err)

	// TODO newebpay
	return &model.SubscriptionCreation{
		Subscription: resp.SubscriptionInfo,
	}, err
}

func (r *mutationResolver) Updatesubscription(ctx context.Context, id string, data *model.SubscriptionUpdateInput) (*model.SubscriptionInfo, error) {
	firebaseID, err := r.GetFirebaseID(ctx)
	if err != nil {
		return nil, err
	}

	_firebaseID, _frequency, err := r.RetrieveExistingSubscriptionFromRemote(ctx, id)
	if err != nil {
		return nil, err
	} else if _firebaseID != firebaseID {
		return nil, fmt.Errorf("you do not have access to this resource, subscription(%s)", id)
	} else if _frequency == model.SubscriptionFrequencyTypeOneTime.String() {
		return nil, fmt.Errorf("%s subscription cannot be updated", _frequency)
	}

	var input *model.SubscriptionPrivateUpdateInput
	if data != nil {
		input = &model.SubscriptionPrivateUpdateInput{
			Desc:          data.Desc,
			NextFrequency: (*model.SubscriptionNextFrequencyType)(data.NextFrequency),
			Note:          data.Note,
			IsCanceled:    data.IsCanceled,
		}

		price, currency, state, err := r.RetrieveMerchandise(ctx, data.NextFrequency.String())
		if err != nil {
			return nil, err
		}
		if state != model.MerchandiseStateTypeActive {
			return nil, fmt.Errorf("frequency(%s) is not %s", model.SubscriptionFrequencyTypeOneTime, model.MerchandiseStateTypeActive)
		}
		amount := int(price)
		input.Amount = &amount
		input.Currency = (*model.SubscriptionCurrencyType)(&currency)
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
	req.Var("input", input)

	var resp struct {
		SubscriptionInfo *model.SubscriptionInfo `json:"updatesubscription"`
	}

	err = r.Client.Run(ctx, req, &resp)

	checkAndPrintGraphQLError(logrus.WithField("mutation", "updatesubscription"), err)

	return resp.SubscriptionInfo, err
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

type mutationResolver struct{ *Resolver }
