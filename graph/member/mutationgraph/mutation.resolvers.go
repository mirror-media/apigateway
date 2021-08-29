package mutationgraph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/mirror-media/apigateway/graph/member/model"
	"github.com/mirror-media/apigateway/graph/member/mutationgraph/generated"
)

func (r *mutationResolver) Createmember(ctx context.Context, data *model.MemberCreateInput) (*model.Member, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) Updatemember(ctx context.Context, id string, data *model.MemberUpdateInput) (*model.Member, error) {
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
