package mutationgraph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"strings"

	graphqlclient "github.com/machinebox/graphql"
	"github.com/mirror-media/apigateway/graph/member/model"
	"github.com/mirror-media/apigateway/graph/member/mutationgraph/generated"
	"github.com/sirupsen/logrus"
)

func (r *mutationResolver) Createmember(ctx context.Context, data *model.MemberCreateInput) (*model.MemberInfo, error) {
	firebaseID, err := r.GetFirebaseID(ctx)
	if err != nil {
		return nil, err
	}

	input := &model.MemberPrivateCreateInput{
		FirebaseID: &firebaseID,
	}
	if data == nil {
		input = nil
	} else {
		input.Address = data.Address
		input.Birthday = data.Birthday
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

	preGQL := []string{"mutation($input: memberCreateInput) {", "createmember(data: $input) {"}

	fieldsOnly := Map(GetPreloads(ctx), func(s string) string {
		ns := strings.Split(s, ".")
		return ns[len(ns)-1]
	})

	preGQL = append(preGQL, fieldsOnly...)
	preGQL = append(preGQL, "}", "}")
	gql := strings.Join(preGQL, "\n")
	req := graphqlclient.NewRequest(gql)
	req.Var("input", input)

	var resp struct {
		Data *struct {
			MemberInfo *model.MemberInfo `json:"member"`
		} `json:"data`
	}

	err = r.Client.Run(ctx, req, &resp)

	checkAndPrintGraphQLError(logrus.WithField("mutation", "createmember"), err)

	return resp.Data.MemberInfo, err
}

func (r *mutationResolver) Updatemember(ctx context.Context, id string, data *model.MemberUpdateInput) (*model.MemberInfo, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) CreateSubscriptionRecurring(ctx context.Context, data *model.SubscriptionRecurringCreateInput) (*model.SubscriptionCreation, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) CreatesSubscriptionOneTime(ctx context.Context, data *model.SubscriptionOneTimeCreateInput) (*model.SubscriptionCreation, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) Updatesubscription(ctx context.Context, id string, data *model.SubscriptionUpdateInput) (*model.Subscription, error) {
	panic(fmt.Errorf("not implemented"))
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

type mutationResolver struct{ *Resolver }
